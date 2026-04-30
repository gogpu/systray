# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report security issues via:

1. **Private Security Advisory** (preferred):
   https://github.com/gogpu/systray/security/advisories/new

2. **GitHub Discussions** (for less critical issues):
   https://github.com/gogpu/gogpu/discussions

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Affected versions
- Potential impact

### Response Timeline

- **Initial Response**: Within 72 hours
- **Fix & Disclosure**: Coordinated with reporter

## Security Considerations

systray uses platform-native APIs via FFI. Users should be aware of:

1. **Native Library Loading** — macOS backend loads ObjC runtime via goffi
2. **D-Bus Communication** — Linux backend communicates over the session D-Bus
3. **Icon Data** — PNG bytes are passed to native APIs for icon conversion
4. **Win32 Callbacks** — Windows backend uses message-only HWND and Shell_NotifyIconW

## Security Contact

- **GitHub Security Advisory**: https://github.com/gogpu/systray/security/advisories/new
- **Public Issues**: https://github.com/gogpu/systray/issues

---

**Thank you for helping keep gogpu/systray secure!**
