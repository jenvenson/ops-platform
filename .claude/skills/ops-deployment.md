# Deployment Management Skill (`ops-deployment`)

## Description
Handle application deployments across different environments.

## Functions
- Trigger application releases
- Monitor deployment progress
- Check deployment status
- Rollback failed deployments
- Archive applications
- View deployment history

## Usage
When you need to deploy applications, check deployment status, or manage application lifecycles, use this skill to interact with the deployment system.

### Examples:
- "Deploy application 'my-app' to environment 'staging'"
- "Check the status of the deployment for 'api-service'"
- "Show me the deployment history for 'frontend-app'"
- "Rollback the last deployment of 'payment-service'"

## Commands
1. **Deployment Operations:**
   - `trigger_deployment(app_id, environment, deploy_type="full")`
   - `get_deployment_status(deployment_id)`
   - `cancel_deployment(deployment_id)`
   - `rollback_deployment(deployment_id)`

2. **Application Management:**
   - `list_applications(project_id="")`
   - `create_application(name, project_id, type)`
   - `archive_application(app_id)`
   - `restore_archived_application(app_id)`

3. **History & Monitoring:**
   - `get_deployment_history(app_id, environment="", limit=20)`
   - `get_active_deployments()`
   - `get_failed_deployments(days_back=7)`
   - `get_deployment_logs(deployment_id)`

4. **Utilities:**
   - `validate_deployment_config(app_id, env_id)`
   - `precheck_deployment(app_id, env_id)`
   - `get_deployment_templates()`

## Integration Notes
This skill interacts with the backend CI/CD module at `/api/deploy/` endpoints and integrates with Jenkins for deployment orchestration. The deployment workflow uses the task manager system defined in `/backend/internal/tasks/task_manager.go`.