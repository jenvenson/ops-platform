# Subagents for OPS Platform

## Overview
Subagents are autonomous agents designed to handle complex operations in the OPS Platform. Each subagent operates independently to perform specific tasks with minimal human intervention.

## 1. Code Review Agent (`code-review-agent`)

### Purpose
Expert code review specialist. Proactively reviews code for quality, security, and maintainability. Use immediately after writing or modifying code.

### Capabilities
- Code quality assessment (structure, readability, performance)
- Security vulnerability detection (SQL injection, XSS, CSRF, etc.)
- Best practices validation
- Code complexity analysis
- Documentation and comments checking
- Code style consistency verification
- Dependency and package security scanning
- Performance optimization suggestions

### Operational Mode
- Runs automatically after code changes are detected
- Performs deep analysis of code quality metrics
- Generates quality reports and recommendations
- Integrates with version control systems for PR reviews

### Triggers
- New code commits
- Pull request creation
- Code modification events
- On-demand quality audits

## 2. Test Validation Agent (`test-validation-agent`)

### Purpose
Specialized Quality Assurance and Test Engineering agent. Mission is to ensure software reliability by designing comprehensive test suites, identifying edge cases, and verifying code correctness through automated testing strategies.

### Capabilities
- Design comprehensive test suites
- Identify boundary conditions and exceptional scenarios
- Verify code correctness and reliability
- Generate unit, integration, and end-to-end tests
- Calculate and optimize test coverage
- Execute automated regression testing
- Perform performance testing verification
- Analyze test results and report failures

### Operational Mode
- Runs continuously to validate code changes
- Automatically generates tests for new features
- Executes regression tests on code changes
- Reports test coverage and quality metrics

### Triggers
- Code modifications
- New feature additions
- Bug fix implementations
- Before deployment validation

## 3. Audit Compliance Agent (`audit-compliance-agent`)

### Purpose
Tracks all system operations, maintains audit trails, ensures compliance with security policies, and generates compliance reports.

### Capabilities
- Track and log all system operations
- Validate compliance with security policies
- Generate compliance reports
- Identify policy violations
- Maintain detailed audit trails
- Monitor access controls and permissions
- Ensure regulatory compliance

### Operational Mode
- Real-time logging of system operations
- Continuous compliance monitoring
- Automated violation detection
- Scheduled compliance reporting

### Triggers
- System operations occurring
- Policy violation detection
- Compliance report generation requests
- Audit trail reviews