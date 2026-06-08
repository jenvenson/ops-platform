# Security Scanning Skill (`ops-security`)

## Description
Manage security scans and vulnerability management.

## Functions
- Create security scanning tasks
- Configure scan targets (CIDR/IP/URL)
- Monitor scan progress
- Generate security reports
- Manage vulnerabilities
- Track and assign vulnerability tickets
- View asset inventory

## Usage
When you need to perform security scans, assess vulnerabilities, or manage security aspects of the infrastructure, use this skill to interact with the security scanning system.

### Examples:
- "Start a security scan for network '192.168.1.0/24'"
- "Show me the progress of security task 'scan-123'"
- "Generate a security report for the latest scan"
- "Create a vulnerability ticket for CVE-2023-1234"
- "List all discovered assets in the 'dmz' network"

## Commands
1. **Scan Management:**
   - `start_scan(target, scan_type="web", web_options={})`
   - `get_scan_progress(task_id)`
   - `cancel_scan(task_id)`
   - `get_scan_results(task_id)`

2. **Asset Management:**
   - `list_assets(network="", service="", limit=50)`
   - `get_asset_details(asset_id)`
   - `update_asset_risk_level(asset_id, risk_level)`

3. **Vulnerability Management:**
   - `list_vulnerabilities(severity="", status="", limit=50)`
   - `get_vulnerability_details(cve_id)`
   - `create_vulnerability_ticket(vuln_id, assignee, due_date)`
   - `update_vulnerability_status(vuln_id, status)`

4. **Ticket Management:**
   - `list_tickets(status="", priority="", limit=20)`
   - `get_ticket_details(ticket_id)`
   - `update_ticket_status(ticket_id, status)`
   - `assign_ticket(ticket_id, assignee)`

5. **Reporting:**
   - `generate_security_report(task_id)`
   - `get_vulnerability_summary()`
   - `export_scan_results(task_id, format="json")`

## Integration Notes
This skill interacts with the security module at `/api/security/` endpoints and uses the scanning engine implemented in `/backend/internal/security/engine.go`. The system integrates Nmap and Nuclei for comprehensive vulnerability scanning and maintains a vulnerability database for tracking and reporting.