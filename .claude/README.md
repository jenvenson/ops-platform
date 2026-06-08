# Claude Skills and Subagents for OPS Platform

This repository contains the Claude skills and subagents designed to enhance the operations of the OPS Platform. These tools provide intelligent automation and management capabilities for the comprehensive operations management system.

## Skills Overview

The following skills are available to manage different aspects of the OPS Platform:

1. **CMDB Management** (`ops-cmdb`) - Manage configuration items in the Configuration Management Database
2. **Deployment Management** (`ops-deployment`) - Handle application deployments across environments
3. **Jenkins Management** (`ops-jenkins`) - Manage Jenkins CI/CD pipelines and views
4. **Consul Management** (`ops-consul`) - Manage Consul key-value configurations and service discovery
5. **Security Scanning** (`ops-security`) - Manage security scans and vulnerability management
6. **Monitoring** (`ops-monitoring`) - Handle system monitoring and metrics collection
7. **Alerting** (`ops-alerting`) - Manage alerts, notifications, and contact information
8. **System Administration** (`ops-admin`) - Handle user management, roles, and system settings

## Subagents Overview

The following subagents provide autonomous operation capabilities:

1. **Security Analysis Agent** - Analyzes security scanning results and manages vulnerabilities
2. **Infrastructure Health Agent** - Monitors infrastructure health and predicts issues
3. **Deployment Validation Agent** - Validates deployments before execution
4. **Consul Config Sync Agent** - Synchronizes Consul configurations across environments
5. **Performance Optimization Agent** - Analyzes performance and suggests optimizations
6. **Audit Compliance Agent** - Tracks operations and ensures compliance

## How to Use

When working with the OPS Platform, you can invoke any of these skills by referencing them directly. For example:

- To manage CMDB items: "Use ops-cmdb to add a new server"
- To trigger a deployment: "Use ops-deployment to deploy my-app to staging"
- To start a security scan: "Use ops-security to scan the 192.168.1.0/24 network"

For ongoing operations that require continuous monitoring, the subagents operate autonomously and provide reports and alerts as needed.

## Architecture Integration

All skills integrate with the existing OPS Platform architecture:
- Backend API endpoints: `/api/cmdb/`, `/api/deploy/`, `/api/security/`, etc.
- Authentication system using JWT tokens
- Database models defined in `/backend/internal/models/`
- Task management system for asynchronous operations
- Audit logging for all operations

## Development Guidelines

When extending these skills or creating new ones:
- Follow the existing structure and patterns
- Ensure proper error handling and validation
- Integrate with the existing audit logging system
- Respect user permissions and access controls
- Maintain consistent response formats

## Maintenance

Regular review of skills and subagents should occur to:
- Update integration points as the platform evolves
- Optimize performance and reliability
- Enhance capabilities based on user feedback
- Ensure security and compliance requirements are met