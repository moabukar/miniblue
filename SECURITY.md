# Security Policy

## Important Note

miniblue is a **local development tool**. It is not designed for production use and should never be exposed to the internet.

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue
2. Email: devopsbymo@gmail.com
3. Include a description of the vulnerability and steps to reproduce

We will respond within 48 hours and work with you to understand and address the issue.

## Scope

Since miniblue is a local emulator, security concerns are primarily around:

- Preventing unintended network exposure
- Ensuring mock tokens cannot be confused with real Azure tokens
- Safe handling of user-provided data in the in-memory store
