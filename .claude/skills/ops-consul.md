# Consul Management Skill (`ops-consul`)

## Description
Manage Consul key-value configurations and service discovery.

## Functions
- Create/update Consul configurations
- Set default configurations
- Test Consul connections
- Batch copy configurations
- Advanced replacement rule processing
- Operation history tracking

## Usage
When you need to manage Consul configurations, perform KV operations, or handle service discovery setup, use this skill to interact with the Consul management system.

### Examples:
- "Create a Consul configuration for 'my-service'"
- "Test connection to Consul server at '10.0.0.1:8500'"
- "Copy Consul configs from 'project-a' to 'project-b' with replacements"
- "List operation history for Consul configurations"

## Commands
1. **Configuration Management:**
   - `create_config(name, address, dc, is_default=false)`
   - `update_config(config_id, params)`
   - `set_as_default(config_id)`
   - `test_connection(config_id)`

2. **Batch Operations:**
   - `batch_copy_configs(source_project, target_projects, replacements={})`
   - `delete_keys_by_suffix(suffix_pattern)`
   - `get_keys_by_suffix(suffix_pattern)`

3. **KV Operations:**
   - `get_kv(key)`
   - `put_kv(key, value)`
   - `delete_kv(key)`
   - `list_kvs(prefix="")`

4. **History & Monitoring:**
   - `get_operation_history(limit=20)`
   - `delete_operation_history(op_id)`
   - `get_config_details(config_id)`

5. **Advanced Features:**
   - `apply_advanced_replacements(replacements)`
   - `export_config_template(config_id)`
   - `import_config_from_template(template)`

## Integration Notes
This skill interacts with the Consul management module at `/api/consul/` endpoints and uses the service defined in `/backend/internal/consul/service.go`. The system maintains operation history for audit purposes and supports advanced replacement rules for complex configuration transformations.