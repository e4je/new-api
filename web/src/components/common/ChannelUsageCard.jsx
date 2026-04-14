import React, { useEffect, useState } from 'react';
import { Card, Tag, Spin, Empty, Tooltip } from '@douyinfe/semi-ui';
import { API, showError } from '../../helpers';
import { useTranslation } from 'react-i18next';

export default function ChannelUsageCard() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [channels, setChannels] = useState([]);

  useEffect(() => {
    fetchChannelUsage();
  }, []);

  const fetchChannelUsage = async () => {
    try {
      const res = await API.get('/api/channel/usage');
      if (res.data.success) {
        setChannels(res.data.data || []);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error.message);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <Card>
        <div className='flex justify-center py-4'>
          <Spin />
        </div>
      </Card>
    );
  }

  if (channels.length === 0) {
    return (
      <Card>
        <Empty
          image={<Empty.defaultIllustration />}
          description={t('暂无渠道用量信息')}
        />
      </Card>
    );
  }

  return (
    <Card title={t('渠道信息')}>
      <div className='grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3'>
        {channels.map((channel) => (
          <div
            key={channel.id}
            className='flex flex-col gap-2 p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors'
          >
            <div className='font-medium text-gray-800 truncate'>
              {channel.name}
            </div>
            <div className='flex flex-wrap gap-1.5'>
              {channel.hourly_limit > 0 && (
                <Tooltip content={t('每小时调用限制剩余')}>
                  <Tag
                    color={channel.hourly_remaining > 0 ? 'orange' : 'red'}
                    type='light'
                    shape='circle'
                    size='small'
                  >
                    {t('时')}:{channel.hourly_remaining}/{channel.hourly_limit}
                  </Tag>
                </Tooltip>
              )}
              {channel.daily_limit > 0 && (
                <Tooltip content={t('每天调用限制剩余')}>
                  <Tag
                    color={channel.daily_remaining > 0 ? 'orange' : 'red'}
                    type='light'
                    shape='circle'
                    size='small'
                  >
                    {t('天')}:{channel.daily_remaining}/{channel.daily_limit}
                  </Tag>
                </Tooltip>
              )}
              {channel.weekly_limit > 0 && (
                <Tooltip content={t('每周调用限制剩余')}>
                  <Tag
                    color={channel.weekly_remaining > 0 ? 'orange' : 'red'}
                    type='light'
                    shape='circle'
                    size='small'
                  >
                    {t('周')}:{channel.weekly_remaining}/{channel.weekly_limit}
                  </Tag>
                </Tooltip>
              )}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}
