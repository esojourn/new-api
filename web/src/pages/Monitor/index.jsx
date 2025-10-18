/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import {
  Button,
  Table,
  Dialog,
  Form,
  Space,
  Tag,
  Spin,
  Empty,
  Popconfirm,
  Input,
  Select,
  InputNumber,
  Collapse,
  Modal,
  Divider
} from '@douyinfe/semi-ui';
import { IconDelete, IconEdit, IconPlayCircle } from '@douyinfe/semi-icons';
import { API, showError, showSuccess, showWarning } from '../../helpers';
import { useTranslation } from 'react-i18next';

export default function Monitor() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [tasks, setTasks] = useState([]);
  const [visible, setVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [editingTask, setEditingTask] = useState(null);
  const [detailTask, setDetailTask] = useState(null);
  const [results, setResults] = useState([]);
  const [resultLoading, setResultLoading] = useState(false);
  const [formApi, setFormApi] = useState(null);
  const [channels, setChannels] = useState([]);
  const [models, setModels] = useState([]);

  // Load monitor tasks
  const loadTasks = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/monitor/tasks');
      if (res && res.data && res.data.data) {
        setTasks(res.data.data.tasks || []);
      }
    } catch (error) {
      showError(t('获取监控任务失败'));
    } finally {
      setLoading(false);
    }
  };

  // Load channels and models for form
  const loadChannelsAndModels = async () => {
    try {
      const channelsRes = await API.get('/api/channels?limit=100');
      if (channelsRes && channelsRes.data) {
        setChannels(channelsRes.data.data || []);
      }

      const modelsRes = await API.get('/api/models?limit=100');
      if (modelsRes && modelsRes.data) {
        setModels(modelsRes.data.data || []);
      }
    } catch (error) {
      showError(t('获取渠道或模型列表失败'));
    }
  };

  // Load task results
  const loadTaskResults = async (taskId) => {
    setResultLoading(true);
    try {
      const res = await API.get(`/api/monitor/tasks/${taskId}/latest-results`);
      if (res && res.data && res.data.data) {
        setResults(res.data.data);
      }
    } catch (error) {
      showError(t('获取监控结果失败'));
    } finally {
      setResultLoading(false);
    }
  };

  useEffect(() => {
    loadTasks();
    loadChannelsAndModels();
  }, []);

  // Handle create/edit task
  const handleSaveTask = async (formValues) => {
    try {
      const payload = {
        name: formValues.name,
        enabled: formValues.enabled || false,
        channels: formValues.channels || [],
        models: formValues.models || [],
        schedule: { interval: formValues.interval || 900 },
        test_content: formValues.test_content || 'hi',
        expected_pattern: formValues.expected_pattern || '',
        max_retries: formValues.max_retries || 2,
        timeout: formValues.timeout || 30,
        remark: formValues.remark || ''
      };

      if (editingTask) {
        // Update existing task
        const res = await API.put(`/api/monitor/tasks/${editingTask.id}`, payload);
        if (res) {
          showSuccess(t('更新监控任务成功'));
          loadTasks();
        }
      } else {
        // Create new task
        const res = await API.post('/api/monitor/tasks', payload);
        if (res) {
          showSuccess(t('创建监控任务成功'));
          loadTasks();
        }
      }
      setVisible(false);
      setEditingTask(null);
    } catch (error) {
      showError(editingTask ? t('更新监控任务失败') : t('创建监控任务失败'));
    }
  };

  // Handle delete task
  const handleDeleteTask = async (taskId) => {
    try {
      await API.delete(`/api/monitor/tasks/${taskId}`);
      showSuccess(t('删除监控任务成功'));
      loadTasks();
    } catch (error) {
      showError(t('删除监控任务失败'));
    }
  };

  // Handle toggle task
  const handleToggleTask = async (taskId, enabled) => {
    try {
      await API.patch(`/api/monitor/tasks/${taskId}/toggle`, { enabled: !enabled });
      showSuccess(!enabled ? t('启用成功') : t('禁用成功'));
      loadTasks();
    } catch (error) {
      showError(t('��换任务状态失败'));
    }
  };

  // Handle run task now
  const handleRunNow = async (taskId) => {
    try {
      setResultLoading(true);
      await API.post(`/api/monitor/tasks/${taskId}/run-now`);
      showSuccess(t('任务已执行'));
      await loadTaskResults(taskId);
    } catch (error) {
      showError(t('执行任务失败'));
    } finally {
      setResultLoading(false);
    }
  };

  // Handle view details
  const handleViewDetails = (task) => {
    setDetailTask(task);
    setDetailVisible(true);
    loadTaskResults(task.id);
  };

  // Handle edit
  const handleEdit = (task) => {
    setEditingTask(task);
    setVisible(true);
    if (formApi) {
      const channels = task.channels ? JSON.parse(task.channels) : [];
      const models = task.models ? JSON.parse(task.models) : [];
      const schedule = task.schedule ? JSON.parse(task.schedule) : { interval: 900 };
      formApi.setValues({
        name: task.name,
        enabled: task.enabled,
        channels: channels,
        models: models,
        interval: schedule.interval || 900,
        test_content: task.test_content,
        expected_pattern: task.expected_pattern,
        max_retries: task.max_retries,
        timeout: task.timeout,
        remark: task.remark
      });
    }
  };

  // Handle create new
  const handleCreate = () => {
    setEditingTask(null);
    setVisible(true);
    if (formApi) {
      formApi.reset();
    }
  };

  const getStatusTag = (status) => {
    const statusMap = {
      0: { color: 'green', text: t('成功') },
      1: { color: 'red', text: t('失败') },
      2: { color: 'orange', text: t('运行中') },
      '-1': { color: 'gray', text: t('未运行') }
    };
    const s = statusMap[status] || statusMap['-1'];
    return <Tag color={s.color}>{s.text}</Tag>;
  };

  const taskColumns = [
    {
      title: t('任务名称'),
      dataIndex: 'name',
      width: 150
    },
    {
      title: t('状态'),
      dataIndex: 'enabled',
      width: 80,
      render: (text, record) => (
        <Tag color={record.enabled ? 'green' : 'gray'}>
          {record.enabled ? t('启用') : t('禁用')}
        </Tag>
      )
    },
    {
      title: t('最后运行'),
      dataIndex: 'last_run_status',
      width: 100,
      render: (text) => getStatusTag(text)
    },
    {
      title: t('平均响应时间'),
      dataIndex: 'avg_response_time',
      width: 120,
      render: (text) => text ? `${text}ms` : '-'
    },
    {
      title: t('操作'),
      width: 200,
      render: (text, record) => (
        <Space>
          <Button
            size='small'
            onClick={() => handleViewDetails(record)}
          >
            {t('详情')}
          </Button>
          <Button
            size='small'
            icon={<IconPlayCircle />}
            onClick={() => handleRunNow(record.id)}
          >
            {t('运行')}
          </Button>
          <Button
            size='small'
            icon={<IconEdit />}
            onClick={() => handleEdit(record)}
          >
            {t('编辑')}
          </Button>
          <Popconfirm
            title={t('删除任务')}
            content={t('确定要删除此监控任务吗？')}
            onConfirm={() => handleDeleteTask(record.id)}
          >
            <Button size='small' icon={<IconDelete />} type='danger'>
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      )
    }
  ];

  return (
    <div className='mt-[60px] px-4'>
      <Spin spinning={loading}>
        <div className='mb-4'>
          <Button onClick={handleCreate} type='primary'>
            {t('创建监控任务')}
          </Button>
        </div>

        {tasks.length === 0 ? (
          <Empty description={t('暂��监控任务')} />
        ) : (
          <Table
            dataSource={tasks}
            columns={taskColumns}
            pagination={false}
            key='id'
          />
        )}

        {/* Create/Edit Task Modal */}
        <Dialog
          title={editingTask ? t('编辑监控任务') : t('创建监控任务')}
          visible={visible}
          onCancel={() => setVisible(false)}
          onOk={() => {
            if (formApi) {
              handleSaveTask(formApi.getValues());
            }
          }}
          width={700}
        >
          <Form
            getFormApi={(api) => setFormApi(api)}
            onSubmit={handleSaveTask}
            initValues={{
              enabled: false,
              interval: 900,
              test_content: 'hi',
              max_retries: 2,
              timeout: 30
            }}
          >
            <Form.Input
              field='name'
              label={t('任务名称')}
              placeholder={t('输入监控任务名称')}
              rules={[{ required: true, message: t('请输入任务名称') }]}
            />

            <Form.Select
              field='channels'
              label={t('监控渠道')}
              placeholder={t('选择要监控的渠道')}
              multiple
              style={{ width: '100%' }}
              optionLabelProp='label'
            >
              {channels.map((channel) => (
                <Select.Option key={channel.id} value={channel.id} label={channel.name}>
                  {channel.name}
                </Select.Option>
              ))}
            </Form.Select>

            <Form.Select
              field='models'
              label={t('监控模型')}
              placeholder={t('输入要监控的模型名称')}
              multiple
              style={{ width: '100%' }}
              maxTagCount={3}
            >
              {models.map((model) => (
                <Select.Option key={model.id} value={model.model_name}>
                  {model.model_name}
                </Select.Option>
              ))}
            </Form.Select>

            <Form.InputNumber
              field='interval'
              label={t('测试间隔（秒）')}
              placeholder={t('输入测试间隔')}
              min={60}
              max={86400}
            />

            <Form.TextArea
              field='test_content'
              label={t('测试内容')}
              placeholder={t('输入测试对话内容')}
              rows={3}
            />

            <Form.Input
              field='expected_pattern'
              label={t('预期响应（正则表达式，可选）')}
              placeholder={t('输入响应验证的正则表达式')}
            />

            <Form.InputNumber
              field='max_retries'
              label={t('最大重试次数')}
              min={0}
              max={10}
            />

            <Form.InputNumber
              field='timeout'
              label={t('超时时间（秒）')}
              min={10}
              max={300}
            />

            <Form.Input
              field='remark'
              label={t('备注')}
              placeholder={t('输入任务备注')}
            />

            <Form.Checkbox field='enabled' label={t('启用此任务')} />
          </Form>
        </Dialog>

        {/* Task Details Modal */}
        <Modal
          title={detailTask?.name}
          visible={detailVisible}
          onCancel={() => setDetailVisible(false)}
          width={900}
          footer={null}
        >
          <Spin spinning={resultLoading}>
            {detailTask && (
              <>
                <Divider>{t('任务信息')}</Divider>
                <div className='grid grid-cols-2 gap-4 mb-4'>
                  <div>
                    <strong>{t('任务名称')}:</strong> {detailTask.name}
                  </div>
                  <div>
                    <strong>{t('状态')}:</strong>
                    <Tag color={detailTask.enabled ? 'green' : 'gray'} className='ml-2'>
                      {detailTask.enabled ? t('启用') : t('禁用')}
                    </Tag>
                  </div>
                  <div>
                    <strong>{t('总运行次数')}:</strong> {detailTask.total_runs}
                  </div>
                  <div>
                    <strong>{t('成功次数')}:</strong> {detailTask.total_successes}
                  </div>
                  <div>
                    <strong>{t('失败次数')}:</strong> {detailTask.total_failures}
                  </div>
                  <div>
                    <strong>{t('平均响应时间')}:</strong> {detailTask.avg_response_time}ms
                  </div>
                </div>

                <Divider>{t('最近测试结果')}</Divider>
                <Table
                  dataSource={results}
                  columns={[
                    {
                      title: t('渠道ID'),
                      dataIndex: 'channel_id',
                      width: 80
                    },
                    {
                      title: t('模型'),
                      dataIndex: 'model',
                      width: 120
                    },
                    {
                      title: t('状态'),
                      dataIndex: 'status',
                      width: 80,
                      render: (status) => getStatusTag(status)
                    },
                    {
                      title: t('响应时间'),
                      dataIndex: 'response_time',
                      width: 100,
                      render: (time) => `${time}ms`
                    },
                    {
                      title: t('错误信息'),
                      dataIndex: 'error_message',
                      render: (text) => (
                        <span title={text} className='truncate max-w-xs inline-block'>
                          {text || '-'}
                        </span>
                      )
                    }
                  ]}
                  pagination={{ pageSize: 10 }}
                  key='id'
                />
              </>
            )}
          </Spin>
        </Modal>
      </Spin>
    </div>
  );
}
