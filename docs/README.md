# Atlas CLI Documentation

## API Specifications

This directory contains API specifications and documentation for Atlassian services.

### Directory Structure

```
docs/
├── README.md                    # This file
├── api-specs/                   # API specifications
│   ├── bitbucketserver.906.postman.json  # Bitbucket Server 9.06 Postman collection
│   └── (future OpenAPI specs)
├── examples/                    # Usage examples
└── guides/                      # Setup and configuration guides
```

### API Specs

- **Bitbucket Server 9.06**: Postman collection with comprehensive REST API endpoints
- **JIRA REST API**: Uses standard JIRA Server REST API v2/v3
- **Confluence REST API**: Uses standard Confluence Server REST API

### Usage

Import the Postman collections into Postman for:
- API exploration and testing
- Understanding request/response formats
- Debugging atlas-cli implementations
- Developing new features

### Authentication

All API specs assume Bearer token or Basic authentication as configured in atlas-cli.

### Reference Links

- [Bitbucket Server REST API](https://developer.atlassian.com/server/bitbucket/rest/)
- [JIRA Server REST API](https://developer.atlassian.com/server/jira/platform/rest-apis/)
- [Confluence Server REST API](https://developer.atlassian.com/server/confluence/rest/api/)