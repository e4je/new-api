package model

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Run against disposable databases with TEST_MYSQL_DSN/TEST_POSTGRES_DSN.
// Optional TEST_MYSQL_LOG_DSN/TEST_POSTGRES_LOG_DSN select a separate log
// database; otherwise logs still use their own connection and isolated tables.
// Do not parallelize: production Task and Log methods use package-level DBs.
func TestTaskLogDatabaseIntegration(t *testing.T) {
	for _, backend := range []struct {
		name   string
		dbType common.DatabaseType
		dsnEnv string
		logEnv string
	}{
		{name: "sqlite", dbType: common.DatabaseTypeSQLite},
		{name: "mysql", dbType: common.DatabaseTypeMySQL, dsnEnv: "TEST_MYSQL_DSN", logEnv: "TEST_MYSQL_LOG_DSN"},
		{name: "postgres", dbType: common.DatabaseTypePostgreSQL, dsnEnv: "TEST_POSTGRES_DSN", logEnv: "TEST_POSTGRES_LOG_DSN"},
	} {
		t.Run(backend.name, func(t *testing.T) {
			dsn := ":memory:"
			if backend.dsnEnv != "" {
				dsn = strings.TrimSpace(os.Getenv(backend.dsnEnv))
				if dsn == "" {
					t.Skip(backend.dsnEnv + " is not configured; real database verification was not run")
				}
			}
			logDSN := strings.TrimSpace(os.Getenv(backend.logEnv))
			if logDSN == "" {
				logDSN = dsn
			}
			prefix := "it_" + strings.ReplaceAll(uuid.NewString(), "-", "") + "_"
			mainDB := openTaskLogIntegrationDatabase(t, backend.dbType, dsn, prefix+"main_")
			logDB := openTaskLogIntegrationDatabase(t, backend.dbType, logDSN, prefix+"log_")

			// CreateTable never upgrades a pre-existing table. Cleanup is limited
			// to the exact randomly named table this fixture created.
			for _, table := range []struct {
				db    *gorm.DB
				model any
				name  string
			}{
				{db: mainDB, model: &Task{}, name: prefix + "main_tasks"},
				{db: logDB, model: &Log{}, name: prefix + "log_logs"},
			} {
				require.False(t, table.db.Migrator().HasTable(table.name), "refusing to reuse an existing table")
				t.Cleanup(func() {
					assert.NoError(t, table.db.Migrator().DropTable(table.name))
				})
				require.NoError(t, table.db.Migrator().CreateTable(table.model))
			}

			originalDB, originalLogDB := DB, LOG_DB
			originalMainType, originalLogType := common.MainDatabaseType(), common.LogDatabaseType()
			DB, LOG_DB = mainDB, logDB
			common.SetDatabaseTypes(backend.dbType, backend.dbType)
			initCol()
			t.Cleanup(func() {
				DB, LOG_DB = originalDB, originalLogDB
				common.SetDatabaseTypes(originalMainType, originalLogType)
				initCol()
			})

			t.Run("private_data_roundtrip", testTaskPrivateDataDatabaseRoundtrip)
			t.Run("private_state_updates_and_cas", testTaskPrivateStateDatabaseUpdates)
			t.Run("separate_log_connection_visibility", testLogOtherDatabaseVisibility)
		})
	}
}

func openTaskLogIntegrationDatabase(t *testing.T, dbType common.DatabaseType, dsn, tablePrefix string) *gorm.DB {
	t.Helper()
	var dialector gorm.Dialector
	switch dbType {
	case common.DatabaseTypeSQLite:
		dialector = sqlite.Open(dsn)
	case common.DatabaseTypeMySQL:
		dialector = mysql.Open(dsn)
	case common.DatabaseTypePostgreSQL:
		dialector = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
	default:
		t.Fatalf("unsupported integration database type %q", dbType)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: tablePrefix},
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	// Connection errors may echo the DSN. Do not include them in test output.
	require.True(t, err == nil, "could not connect to %s test database; check its test DSN and server", dbType)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	query := "SELECT version()"
	if dbType == common.DatabaseTypeSQLite {
		query = "SELECT sqlite_version()"
	}
	var version string
	require.NoError(t, db.Raw(query).Scan(&version).Error)
	require.NotEmpty(t, version)
	t.Logf("%s %s: %s", dbType, tablePrefix, version)
	return db
}

func testTaskPrivateDataDatabaseRoundtrip(t *testing.T) {
	for _, test := range []struct {
		name       string
		private    TaskPrivateData
		legacyJSON string
	}{
		{name: "empty"},
		{
			name:       "legacy_json",
			private:    TaskPrivateData{Key: "fixture-key", UpstreamTaskID: "legacy-upstream", TokenId: 7, ResponsesBackground: true},
			legacyJSON: `{"key":"fixture-key","upstream_task_id":"legacy-upstream","token_id":7,"responses_background":true}`,
		},
		{name: "plugin_state_only", private: TaskPrivateData{PluginState: json.RawMessage(`{"cursor":"round-1","zero":0,"enabled":false}`)}},
		{name: "poll_failures_only", private: TaskPrivateData{PollFailures: 3}},
	} {
		t.Run(test.name, func(t *testing.T) {
			task := &Task{TaskID: test.name, Status: TaskStatusInProgress, PrivateData: test.private}
			if test.legacyJSON != "" {
				task.PrivateData = TaskPrivateData{}
			}
			require.NoError(t, DB.Create(task).Error)
			if test.legacyJSON != "" {
				require.NoError(t, DB.Model(task).Update("private_data", test.legacyJSON).Error)
			}
			var loaded Task
			require.NoError(t, DB.First(&loaded, task.ID).Error)
			state, wantState := loaded.PrivateData.PluginState, test.private.PluginState
			loaded.PrivateData.PluginState = nil
			want := test.private
			want.PluginState = nil
			assert.Equal(t, want, loaded.PrivateData)
			if len(wantState) == 0 {
				assert.Empty(t, state)
			} else {
				assert.JSONEq(t, string(wantState), string(state))
			}
			if test.name == "empty" {
				var count int64
				require.NoError(t, DB.Model(&Task{}).Where("id = ? AND private_data IS NULL", task.ID).Count(&count).Error)
				assert.EqualValues(t, 1, count, "empty private state must retain SQL NULL semantics")
			}
		})
	}
}

func testTaskPrivateStateDatabaseUpdates(t *testing.T) {
	task := &Task{
		TaskID: "state-update", Status: TaskStatusInProgress, Progress: "50%", Quota: 1000,
		PrivateData: TaskPrivateData{Key: "fixture-key", PluginState: json.RawMessage(`{"cursor":"old"}`), PollFailures: 1},
	}
	require.NoError(t, DB.Create(task).Error)
	task.PrivateData.PluginState = json.RawMessage(`{"cursor":"new"}`)
	task.PrivateData.PollFailures = 4
	won, err := task.UpdateWithStatus(TaskStatusInProgress)
	require.NoError(t, err)
	require.True(t, won)
	var loaded Task
	require.NoError(t, DB.First(&loaded, task.ID).Error)
	assert.EqualValues(t, TaskStatusInProgress, loaded.Status)
	assert.Equal(t, "50%", loaded.Progress)
	assert.Equal(t, 1000, loaded.Quota)
	assert.Equal(t, "fixture-key", loaded.PrivateData.Key)
	assert.JSONEq(t, `{"cursor":"new"}`, string(loaded.PrivateData.PluginState))
	assert.Equal(t, 4, loaded.PrivateData.PollFailures)

	loaded.PrivateData.PluginState = nil
	loaded.PrivateData.PollFailures = 0
	won, err = loaded.UpdateWithStatus(TaskStatusInProgress)
	require.NoError(t, err)
	require.True(t, won)
	var cleared Task
	require.NoError(t, DB.First(&cleared, task.ID).Error)
	assert.Empty(t, cleared.PrivateData.PluginState)
	assert.Zero(t, cleared.PrivateData.PollFailures)
	assert.Equal(t, "fixture-key", cleared.PrivateData.Key)

	// Two workers that read the same nonterminal state must not both win a
	// terminal transition, even if the second one reports a different result.
	stale := cleared
	cleared.Status = TaskStatusFailure
	cleared.FailReason = "poll failure cutoff"
	cleared.PrivateData.PollFailures = 5
	won, err = cleared.UpdateWithStatus(TaskStatusInProgress)
	require.NoError(t, err)
	require.True(t, won)
	stale.Status = TaskStatusSuccess
	won, err = stale.UpdateWithStatus(TaskStatusInProgress)
	require.NoError(t, err)
	assert.False(t, won)
	var final Task
	require.NoError(t, DB.First(&final, task.ID).Error)
	assert.EqualValues(t, TaskStatusFailure, final.Status)
	assert.Equal(t, "poll failure cutoff", final.FailReason)
	assert.Equal(t, 5, final.PrivateData.PollFailures)
	assert.Equal(t, 1000, final.Quota)
	var count int64
	require.NoError(t, DB.Model(&Task{}).Where("task_id = ?", task.TaskID).Count(&count).Error)
	assert.EqualValues(t, 1, count, "a losing CAS must not insert another task")
}

func testLogOtherDatabaseVisibility(t *testing.T) {
	other := NewLogOther()
	require.True(t, other.SetPublic("request_path", "/v1/responses"))
	require.True(t, other.SetPublic("upstream_model_name", "private-model"))
	require.True(t, other.SetPublic("counter", uint64(9007199254740993)))
	require.True(t, other.SetAdmin("reject_reason", "fixture-reject"))
	require.True(t, other.SetAudit("method", "POST"))
	require.True(t, other.SetRoot("upstream_request_id", "fixture-root"))
	for _, test := range []struct {
		name   string
		stored string
		user   string
		admin  string
		root   string
	}{
		{
			name: "scoped", stored: other.JSONString(),
			user:  `{"request_path":"/v1/responses","counter":9007199254740993}`,
			admin: `{"request_path":"/v1/responses","counter":9007199254740993,"upstream_model_name":"private-model","admin_info":{"reject_reason":"fixture-reject"},"audit_info":{"method":"POST"}}`,
			root:  other.JSONString(),
		},
		{
			name:   "legacy",
			stored: `{"request_path":"/v1/responses","channel_id":17,"channel_name":"private-channel","reject_reason":"legacy-reject","upstream_model_name":"private-model","admin_info":{"existing":true},"audit_info":{"method":"POST"},"root_info":{"upstream_request_id":"fixture-root"}}`,
			user:   `{"request_path":"/v1/responses"}`,
			admin:  `{"request_path":"/v1/responses","channel_id":17,"channel_name":"private-channel","upstream_model_name":"private-model","admin_info":{"existing":true,"reject_reason":"legacy-reject"},"audit_info":{"method":"POST"}}`,
			root:   `{"request_path":"/v1/responses","channel_id":17,"channel_name":"private-channel","upstream_model_name":"private-model","admin_info":{"existing":true,"reject_reason":"legacy-reject"},"audit_info":{"method":"POST"},"root_info":{"upstream_request_id":"fixture-root"}}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			stored := &Log{UserId: 19, Type: LogTypeConsume, Other: test.stored}
			require.NoError(t, createLog(stored))
			for _, role := range []struct {
				name string
				want string
			}{
				{name: "user", want: test.user},
				{name: "admin", want: test.admin},
				{name: "root", want: test.root},
			} {
				t.Run(role.name, func(t *testing.T) {
					var loaded Log
					require.NoError(t, LOG_DB.First(&loaded, stored.Id).Error)
					switch role.name {
					case "user":
						formatUserLogs([]*Log{&loaded}, 0)
					case "admin":
						FormatAdminLogs([]*Log{&loaded})
					case "root":
						FormatRootLogs([]*Log{&loaded})
					}
					assert.JSONEq(t, role.want, loaded.Other)
					if test.name == "scoped" {
						var fields map[string]json.RawMessage
						require.NoError(t, common.UnmarshalJsonStr(loaded.Other, &fields))
						assert.Equal(t, "9007199254740993", string(fields["counter"]))
					}
				})
			}
			var unchanged Log
			require.NoError(t, LOG_DB.First(&unchanged, stored.Id).Error)
			assert.JSONEq(t, test.stored, unchanged.Other, "role projections must not rewrite the stored log")
		})
	}
}
