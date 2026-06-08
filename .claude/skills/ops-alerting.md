# Alerting Skill (`ops-alerting`)

## Description
Manage alerts, notifications, and contact information.

## Functions
- Create alert rules
- Configure notification channels
- Manage contact information
- Process alert events
- Configure templates
- Track alert status

## Usage
When you need to configure alerting rules, manage notification channels, or handle alert events, use this skill to interact with the alerting system.

### Examples:
- "Create an alert rule for high CPU usage"
- "Configure a Slack notification channel"
- "List all active alert events"
- "Update the notification template for critical alerts"

## Commands
1. **Alert Rules:**
   - `create_alert_rule(name, condition, severity, description)`
   - `update_alert_rule(rule_id, params)`
   - `delete_alert_rule(rule_id)`
   - `list_alert_rules()`

2. **Notification Channels:**
   - `add_notification_channel(type, config)`
   - `update_notification_channel(channel_id, config)`
   - `delete_notification_channel(channel_id)`
   - `list_notification_channels()`

3. **Contact Management:**
   - `add_contact(name, email, phone="", dingtalk_webhook="")`
   - `update_contact(contact_id, params)`
   - `remove_contact(contact_id)`
   - `list_contacts()`

4. **Alert Events:**
   - `list_alert_events(status="active", severity="", limit=20)`
   - `get_alert_event_details(event_id)`
   - `acknowledge_alert(event_id, comment)`
   - `close_alert(event_id, resolution)`

5. **Template Management:**
   - `create_notification_template(name, subject, body)`
   - `update_notification_template(template_id, params)`
   - `get_available_template_vars()`
   - `preview_template(template_id, sample_data)`

## Integration Notes
This skill interacts with the alerting module at `/api/alert/` endpoints and integrates with Prometheus Alertmanager for alert evaluation. The system supports multiple notification channels including email, DingTalk, and webhook integrations with customizable templates.