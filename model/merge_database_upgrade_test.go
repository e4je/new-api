package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestLegacyOptionsPrimaryKeyRepair(t *testing.T) {
	if os.Getenv("MERGE_DB_CONFIRM") != "isolated-local-test" {
		t.Skip("requires explicitly configured disposable databases")
	}
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			dsn := "local"
			if dialect == "mysql" {
				dsn = os.Getenv("TEST_MYSQL_DSN")
			}
			if dialect == "postgres" {
				dsn = os.Getenv("TEST_POSTGRES_DSN")
			}
			require.NotEmpty(t, dsn)
			previousPath := common.SQLitePath
			common.SQLitePath = filepath.Join(t.TempDir(), "options.db")
			t.Cleanup(func() { common.SQLitePath = previousPath })
			t.Setenv("MERGE_REPAIR_DSN", dsn)
			db, _, err := chooseDB("MERGE_REPAIR_DSN", false)
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })
			require.False(t, db.Migrator().HasTable("options"), "requires an empty disposable database")
			type legacyOption struct {
				Key   string `gorm:"type:varchar(191)"`
				Value string `gorm:"type:text"`
			}
			require.NoError(t, db.Table("options").Migrator().CreateTable(&legacyOption{}))
			require.NoError(t, db.Table("options").Create(&[]legacyOption{{Key: "Duplicate", Value: "same"}, {Key: "Duplicate", Value: "same"}, {Key: "Keep", Value: "中文"}}).Error)
			for range 2 {
				require.NoError(t, migrateOptionPrimaryKey(db))
			}
			var options []Option
			require.NoError(t, db.Order(clause.OrderByColumn{Column: clause.Column{Name: "key"}}).Find(&options).Error)
			assert.Equal(t, []Option{{Key: "Duplicate", Value: "same"}, {Key: "Keep", Value: "中文"}}, options)
			assert.Error(t, db.Create(&Option{Key: "Keep", Value: "must not replace"}).Error)
			require.NoError(t, db.Save(&Option{Key: "Keep", Value: "updated"}).Error)
			var updated Option
			require.NoError(t, db.Where(clause.Eq{Column: "key", Value: "Keep"}).First(&updated).Error)
			assert.Equal(t, "updated", updated.Value)
			var count int64
			require.NoError(t, db.Model(&Option{}).Count(&count).Error)
			assert.EqualValues(t, 2, count)
			tables, err := db.Migrator().GetTables()
			require.NoError(t, err)
			var backups []string
			for _, table := range tables {
				if strings.HasPrefix(table, "options_legacy_") {
					backups = append(backups, table)
				}
			}
			require.Len(t, backups, 1)
			require.NoError(t, db.Table(backups[0]).Count(&count).Error)
			assert.EqualValues(t, 3, count)
		})
	}
}

// Opt-in, disposable database fixture. Copy this function and its imports into
// the release revision and run with MERGE_DB_PHASE=seed, then run the candidate
// with phase=verify. The separate repair test requires the newer migration API.
// A fresh database is exercised by seed at the candidate revision as well.
func TestReleaseDatabaseUpgrade(t *testing.T) {
	phase := os.Getenv("MERGE_DB_PHASE")
	if phase == "" {
		t.Skip("requires explicitly configured disposable databases")
	}
	require.Contains(t, []string{"seed", "verify"}, phase)
	require.Equal(t, "isolated-local-test", os.Getenv("MERGE_DB_CONFIRM"))
	root := os.Getenv("MERGE_DB_ROOT")
	require.NotEmpty(t, root)
	oldDB, oldLogDB, oldPath, oldMaster := DB, LOG_DB, common.SQLitePath, common.IsMasterNode
	mainType, logType := common.MainDatabaseType(), common.LogDatabaseType()
	t.Cleanup(func() {
		DB, LOG_DB, common.SQLitePath, common.IsMasterNode = oldDB, oldLogDB, oldPath, oldMaster
		common.SetDatabaseTypes(mainType, logType)
		initCol()
	})
	common.IsMasterNode = true
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			dsn, logDSN := "local", "local"
			if dialect == "mysql" {
				dsn, logDSN = os.Getenv("TEST_MYSQL_DSN"), os.Getenv("TEST_MYSQL_LOG_DSN")
			}
			if dialect == "postgres" {
				dsn, logDSN = os.Getenv("TEST_POSTGRES_DSN"), os.Getenv("TEST_POSTGRES_LOG_DSN")
			}
			require.NotEmpty(t, dsn)
			require.NotEmpty(t, logDSN)
			t.Setenv("SQL_DSN", dsn)
			t.Setenv("LOG_SQL_DSN", logDSN)
			t.Setenv("SQL_MAX_OPEN_CONNS", "5")
			t.Setenv("SQL_MAX_IDLE_CONNS", "2")
			for pass := 0; pass < 2; pass++ {
				common.SQLitePath = filepath.Join(root, "main.db")
				require.NoError(t, InitDB())
				common.SQLitePath = filepath.Join(root, "log.db")
				require.NoError(t, InitLogDB())
				sqlDB, err := DB.DB()
				require.NoError(t, err)
				sqlLog, err := LOG_DB.DB()
				require.NoError(t, err)
				if pass == 0 && phase == "seed" {
					var count int64
					require.NoError(t, DB.Model(&User{}).Count(&count).Error)
					require.Zero(t, count)
					require.NoError(t, DB.Create(&User{Username: "merge_fixture", Password: "synthetic-hash", Quota: 5000000000, UsedQuota: 12345}).Error)
					require.NoError(t, DB.Create(&Channel{Name: "merge_fixture", Models: "fixture-model", Group: "default", UsedQuota: 12345}).Error)
					require.NoError(t, DB.Create(&Token{Key: strings.Repeat("x", 32), Name: "merge_fixture", RemainQuota: 12345}).Error)
					require.NoError(t, DB.Create(&Option{Key: "MergeFixture", Value: "中文-preserved"}).Error)
					require.NoError(t, DB.Create(&SystemTask{TaskID: "merge_fixture", Type: "channel_test", Status: SystemTaskStatusSucceeded, State: "{\"done\":1}"}).Error)
					require.NoError(t, LOG_DB.Create(&Log{Username: "merge_fixture", Content: "preserved usage", Quota: 12345}).Error)
					require.NoError(t, LOG_DB.Create(&AuditLog{EventId: "merge_fixture", Username: "merge_fixture", ActorRole: common.RoleRootUser, Content: "preserved audit"}).Error)
				}
				var user User
				require.NoError(t, DB.Where("username = ?", "merge_fixture").First(&user).Error)
				assert.EqualValues(t, 5000000000, user.Quota)
				assert.EqualValues(t, 12345, user.UsedQuota)
				var channel Channel
				require.NoError(t, DB.Where("name = ?", "merge_fixture").First(&channel).Error)
				assert.EqualValues(t, 12345, channel.UsedQuota)
				var token Token
				require.NoError(t, DB.Where("name = ?", "merge_fixture").First(&token).Error)
				assert.EqualValues(t, 12345, token.RemainQuota)
				var option Option
				require.NoError(t, DB.Where(commonKeyCol+" = ?", "MergeFixture").First(&option).Error)
				assert.Equal(t, "中文-preserved", option.Value)
				var task SystemTask
				require.NoError(t, DB.Where("task_id = ?", "merge_fixture").First(&task).Error)
				assert.Equal(t, "{\"done\":1}", task.State)
				var log Log
				require.NoError(t, LOG_DB.Where("username = ?", "merge_fixture").First(&log).Error)
				assert.EqualValues(t, 12345, log.Quota)
				var audit AuditLog
				require.NoError(t, LOG_DB.Where("event_id = ?", "merge_fixture").First(&audit).Error)
				assert.Equal(t, "preserved audit", audit.Content)
				assert.True(t, DB.Migrator().HasIndex(&Token{}, "idx_tokens_key"))
				assert.True(t, LOG_DB.Migrator().HasIndex(&AuditLog{}, "idx_audit_user_time"))
				assert.Error(t, DB.Create(&User{Username: "merge_fixture"}).Error)
				assert.Error(t, DB.Create(&Token{Key: strings.Repeat("x", 32)}).Error)
				assert.Error(t, DB.Create(&Option{Key: "MergeFixture", Value: "must not overwrite"}).Error)
				assert.Error(t, LOG_DB.Create(&AuditLog{EventId: "merge_fixture"}).Error)
				require.NoError(t, sqlDB.Close())
				require.NoError(t, sqlLog.Close())
			}
		})
	}
}
