# Jenkins Management Skill (`ops-jenkins`)

## Description
Manage Jenkins CI/CD pipelines and views.

## Functions
- Import Jenkins jobs
- Sync job details
- Trigger builds
- Copy views between projects
- Delete jobs/views
- Manage Jenkins credentials
- Handle script approvals

## Usage
When you need to manage Jenkins CI/CD jobs, views, or perform CI/CD operations, use this skill to interact with the Jenkins integration system.

### Examples:
- "Import Jenkins jobs from server 'jenkins-dev'"
- "Trigger a build for job 'my-app-build'"
- "Copy view 'dev-team-view' to 'prod-team-view'"
- "Delete jobs matching pattern 'temp-*'"

## Commands
1. **Job Management:**
   - `import_jobs(jenkins_server_url)`
   - `sync_job_details(job_names)`
   - `trigger_build(job_name, parameters={})`
   - `get_job_status(job_name)`
   - `delete_jobs(pattern)`

2. **View Management:**
   - `copy_view(source_view, target_view, replacements={})`
   - `create_view(view_name, jobs=[])`
   - `delete_view(view_name)`
   - `list_views()`

3. **Credentials Management:**
   - `create_ssh_credential(id, username, private_key)`
   - `update_credential(cred_id, params)`
   - `delete_credential(cred_id)`

4. **Script Management:**
   - `approve_pending_scripts()`
   - `list_pending_scripts()`

5. **Monitoring:**
   - `get_build_history(job_name, limit=10)`
   - `get_running_builds()`
   - `get_build_logs(job_name, build_number)`

## Integration Notes
This skill interacts with the Jenkins API through the client at `/backend/pkg/jenkins/client.go` and uses the CMDB integration for project mapping. The async task system handles bulk operations like view copying and job deletion.