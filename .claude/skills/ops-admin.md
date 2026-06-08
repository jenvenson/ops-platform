# System Administration Skill (`ops-admin`)

## Description
Handle user management, roles, and system settings.

## Functions
- Create/update/delete users
- Manage roles and permissions
- Configure system menus
- Update user profiles
- Reset passwords
- Configure system settings

## Usage
When you need to manage users, roles, permissions, or system configurations, use this skill to interact with the administration system.

### Examples:
- "Create a new user 'john.doe@example.com' with role 'developer'"
- "Update user 'jane.smith' role to 'admin'"
- "Configure system menu items for 'ops' role"
- "Reset password for user 'bob.wilson'"

## Commands
1. **User Management:**
   - `create_user(username, email, role, real_name="", password="")`
   - `update_user(user_id, params)`
   - `delete_user(user_id)`
   - `reset_user_password(user_id, new_password="")`
   - `list_users(role="", limit=50)`

2. **Role Management:**
   - `create_role(name, code, description)`
   - `update_role(role_id, params)`
   - `delete_role(role_id)`
   - `assign_permissions_to_role(role_id, permissions)`
   - `list_roles()`

3. **Menu Management:**
   - `create_menu_item(title, key, path, icon, parent_id=null)`
   - `update_menu_item(menu_id, params)`
   - `delete_menu_item(menu_id)`
   - `get_role_menu(role_id)`
   - `set_role_menu_permissions(role_id, menu_items)`

4. **System Configuration:**
   - `get_system_settings()`
   - `update_system_setting(key, value)`
   - `get_audit_log(start_date, end_date, user_id="")`
   - `export_system_config()`

5. **User Profile:**
   - `get_user_profile(user_id)`
   - `update_user_profile(user_id, profile_data)`
   - `change_user_password(user_id, current_password, new_password)`

## Integration Notes
This skill interacts with the auth and admin modules at `/api/auth/` and `/api/admin/` endpoints. The system uses JWT-based authentication and implements role-based access control (RBAC) for fine-grained permissions. User information is stored in the database with support for real names and role assignments.