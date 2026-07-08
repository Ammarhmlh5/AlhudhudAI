# Security Policy

## Supported Versions

We provide security updates for the following versions:

| Version | Supported |
|---------|-----------|
| Latest  | ✅ Yes    |
| Previous| ⚠️ Limited |
| Older   | ❌ No     |

## Reporting a Vulnerability

We take the security of ALHudhudAI seriously. If you believe you have found a security vulnerability, please report it to us responsibly.

### How to Report

1. **Do not** disclose the vulnerability publicly
2. Email your report to: security@alhudhud.ai
3. Include a detailed description of the vulnerability
4. Include steps to reproduce the issue
5. Include any relevant screenshots or logs

### What to Expect

- We will acknowledge your report within 24 hours
- We will provide an initial assessment within 5 business days
- We will keep you updated on our progress
- We will not take legal action against responsible disclosure

## Security Best Practices

### Environment Variables

Never commit sensitive data to version control. Use environment variables for:

- Database credentials
- API keys and secrets
- JWT secrets
- Redis passwords
- Admin passwords

Example:
```bash
# .env file (never commit this)
DB_PASSWORD=${DB_PASSWORD}
JWT_SECRET=${JWT_SECRET}
```

### Database Security

- Always use SSL/TLS for database connections (`DB_SSLMODE=require`)
- Enable Row Level Security (RLS) on multi-tenant tables
- Use parameterized queries to prevent SQL injection
- Keep database user privileges to minimum required

### API Security

- Use CSRF protection for state-changing requests
- Implement rate limiting to prevent abuse
- Use proper authentication and authorization
- Validate all input data

### Password Security

- Use strong password policies
- Hash passwords with bcrypt (cost factor ≥ 12)
- Implement proper session management
- Use secure session cookies (HttpOnly, Secure, SameSite)

## Security Checklist

- [ ] All environment variables use placeholders, not plaintext
- [ ] Database connections use SSL/TLS
- [ ] Row Level Security is enabled on all multi-tenant tables
- [ ] NOT NULL constraints on critical columns
- [ ] CHECK constraints on financial data
- [ ] Circuit breakers for external service calls
- [ ] Rate limiting on all API endpoints
- [ ] CSRF protection enabled
- [ ] Audit logging for sensitive operations
- [ ] Regular security updates applied

## Related Documentation

- [Environment Variables](docs/environment-variables.md)
- [Database Schema](docs/database-schema.md)
- [Migration Guide](docs/migrations.md)
- [API Documentation](docs/api.md)