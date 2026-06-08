import apiClient from './client';

interface CreateAggregatePackageRequest {
  project_name: string;
  app_names: string[];
  task_name: string;
  description?: string;
}

interface CreateAggregatePackageResponse {
  success: boolean;
  data?: {
    task_id: number;
  };
  error?: string;
}

interface TaskRecord {
  id: number;
  task_name: string;
  project_name: string;
  app_names: string[];
  jenkins_job_name: string;
  jenkins_job_url: string;
  status: string;
  triggered_by: string;
  start_time?: string;
  end_time?: string;
  duration?: number;
  error_message?: string;
  results: Array<{
    app_name: string;
    consul_tag: string;
    status: string;
    jenkins_build_num?: number;
    duration?: number;
    download_url?: string;
    error_message?: string;
  }>;
  created_at: string;
}

interface TaskListResponse {
  success: boolean;
  data: {
    total: number;
    page: number;
    limit: number;
    tasks: TaskRecord[];
  };
}

interface TaskDetailResponse {
  success: boolean;
  data: {
    task: TaskRecord;
  };
}

interface TaskStatus {
  status: string;
  progress: number;
  overall_status: string;
  app_statuses: Array<{
    app_name: string;
    status: string;
  }>;
}

interface TaskStatusResponse {
  success: boolean;
  data: TaskStatus;
}

interface JenkinsJob {
  name: string;
  view: string;
  url: string;
  color: string;
  job_name: string;
}

interface QueryJenkinsJobsResponse {
  success: boolean;
  data: JenkinsJob[];
  total: number;
}

interface QueryJenkinsJobsResponse {
  success: boolean;
  data: JenkinsJob[];
  total: number;
}

interface QueryAppTagsResponse {
  success: boolean;
  data: {
    [key: string]: string; // app_name: tag mapping
  };
  error?: string;
}

interface QueryConsulKvResponse {
  success: boolean;
  data: {
    [key: string]: string; // key-value pairs from Consul
  };
  error?: string;
}

interface ConsulConfig {
  id: number;
  name: string;
  address: string;
  is_default: boolean;
}

interface ConsulConfigsResponse {
  success: boolean;
  data: ConsulConfig[];
}

const aggregatePackageAPI = {
  createTask: (data: CreateAggregatePackageRequest) =>
    apiClient.post<CreateAggregatePackageResponse>('/deploy/aggregate-package', data),

  getTasks: (params?: { project_name?: string; status?: string; start_time?: string; end_time?: string; page?: number; limit?: number }) =>
    apiClient.get<TaskListResponse>('/deploy/aggregate-package', { params }),

  getTask: (taskId: number) =>
    apiClient.get<TaskDetailResponse>(`/deploy/aggregate-package/${taskId}`),

  getTaskStatus: (taskId: number) =>
    apiClient.get<TaskStatusResponse>(`/deploy/aggregate-package/${taskId}/status`),

  queryAppTags: (appNames: string[]) =>
    apiClient.post<QueryAppTagsResponse>('/deploy/query-app-tags', { app_names: appNames }),

  queryJenkinsJobs: () =>
    apiClient.get<QueryJenkinsJobsResponse>('/deploy/query-jenkins-jobs'),

  queryConsulKv: (path: string, options?: { signal?: AbortSignal; consul_config_id?: number }) => {
    const params: any = { path };
    if (options?.consul_config_id) {
      params.consul_config_id = options.consul_config_id;
    }
    const config = options?.signal ? { params, signal: options.signal } : { params };
    return apiClient.get<QueryConsulKvResponse>(`/deploy/query-consul-kv`, config);
  },

  getConsulConfigs: () =>
    apiClient.get<ConsulConfigsResponse>('/deploy/consul-configs'),
};

export { aggregatePackageAPI };
export type { CreateAggregatePackageRequest, TaskRecord, TaskStatus };