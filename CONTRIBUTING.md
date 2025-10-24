# Contributing Guide

Thank you for your interest in contributing to the Saddy project! We welcome all forms of contributions.

## How to Contribute

### Reporting Bugs

If you've found a bug, please:

1. Check [Issues](../../issues) to confirm the issue hasn't been reported
2. If not found, create a new Issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected and actual behavior
   - System environment information (OS, Go version, etc.)
   - Related logs or screenshots

### Suggesting Features

We welcome feature suggestions! Please:

1. Check [Issues](../../issues) and [Discussions](../../discussions) for existing suggestions
2. Create a new Issue with the "enhancement" label including:
   - Clear description of the feature
   - Use case and motivation
   - Possible implementation approach (if known)

### Submitting Code Changes

#### Development Setup

1. **Fork the repository**
   ```bash
   # Fork on GitHub, then clone your fork
   git clone https://github.com/YOUR_USERNAME/saddy.git
   cd saddy
   ```

2. **Add upstream remote**
   ```bash
   git remote add upstream https://github.com/original-owner/saddy.git
   ```

3. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Set up development environment**
   ```bash
   # Install dependencies
   go mod download

   # Run tests
   make test

   # Build the project
   make build
   ```

#### Code Guidelines

- **Go Code Style**: Follow standard Go formatting (`gofmt -w .`)
- **Commit Messages**: Use conventional commits format
  - `feat: add new feature`
  - `fix: resolve bug in component`
  - `docs: update documentation`
  - `refactor: improve code structure`
  - `test: add unit tests`
  - `chore: maintenance tasks`

- **Testing**: Ensure all tests pass
  ```bash
  make test
  make test-integration  # if available
  ```

- **Documentation**: Update relevant documentation
  - README.md for user-facing changes
  - API documentation for API changes
  - Code comments for complex logic

#### Submitting Pull Requests

1. **Update your branch**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run final checks**
   ```bash
   make lint    # if available
   make test
   make build
   ```

3. **Commit and push**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   git push origin feature/your-feature-name
   ```

4. **Create Pull Request**
   - Provide clear description of changes
   - Reference related issues
   - Include screenshots if UI changes
   - Ensure CI checks pass

## Development Workflow

### Branch Strategy

- `main`: Stable production branch
- `develop`: Development branch (if used)
- `feature/*`: Feature branches
- `bugfix/*`: Bug fix branches
- `hotfix/*`: Emergency fixes

### Code Review Process

1. All changes require review
2. At least one maintainer approval
3. Automated tests must pass
4. Documentation updated if needed
5. Follow project coding standards

### Testing

- **Unit Tests**: Test individual functions and packages
- **Integration Tests**: Test component interactions
- **Manual Testing**: Test UI and user workflows
- **Performance Tests**: For performance-critical changes

## Areas for Contribution

### Code

- **Core Features**: Proxy, caching, SSL/TLS management
- **Web Interface**: UI improvements, new components
- **API Endpoints**: New API functionality
- **Performance**: Optimization and improvements
- **Security**: Security enhancements and fixes

### Documentation

- **README**: Installation, usage, and deployment guides
- **API Documentation**: API reference and examples
- **Tutorials**: Step-by-step guides
- **Translation**: Multi-language support

### Tools and Automation

- **CI/CD**: GitHub Actions, build pipelines
- **Testing**: Test frameworks, test coverage
- **Development Tools**: Scripts, utilities
- **Monitoring**: Metrics, logging improvements

## Getting Help

### Resources

- [Documentation](README.md)
- [API Reference](README.md#-rest-api)
- [Configuration Guide](README.md#️-configuration-guide)
- [Troubleshooting](README.md#-troubleshooting)

### Community

- [GitHub Issues](../../issues): Bug reports and feature requests
- [GitHub Discussions](../../discussions): General questions and ideas
- [Discord/Slack]: If available (check README for links)

### Maintainers

- **Primary Maintainer**: [Maintainer Name](https://github.com/maintainer)
- **Contact**: Create an issue or start a discussion

## Contribution Recognition

### Contributors

All contributors are recognized in:
- README.md contributors section
- Release notes for significant contributions
- GitHub contributor statistics

### Types of Contributions

- **Code**: Bug fixes, features, performance improvements
- **Documentation**: Guides, tutorials, API docs
- **Design**: UI/UX improvements, graphics
- **Testing**: Test cases, bug reporting
- **Community**: Support, feedback, ideas

## Code of Conduct

Please be respectful and constructive in all interactions. See our [Code of Conduct](CODE_OF_CONDUCT.md) for detailed guidelines.

## Release Process

### Version Management

- Follow [Semantic Versioning](https://semver.org/)
- Update CHANGELOG.md for all changes
- Tag releases with version numbers

### Release Checklist

- [ ] All tests passing
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Version numbers updated
- [ ] Release notes prepared
- [ ] Security review (if applicable)

## License

By contributing to this project, you agree that your contributions will be licensed under the same license as the project (MIT License).

---

## Quick Start for Contributors

```bash
# 1. Fork and clone
git clone https://github.com/YOUR_USERNAME/saddy.git
cd saddy

# 2. Set up development environment
go mod download
make build

# 3. Make your changes
# ... edit files ...

# 4. Test your changes
make test

# 5. Commit and submit PR
git add .
git commit -m "feat: describe your changes"
git push origin feature/your-feature-name
# Then create a PR on GitHub
```

Thank you for contributing to Saddy! 🚀