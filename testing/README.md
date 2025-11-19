# E2E Tests

This directory contains end-to-end (E2E) tests for the project. These tests are designed to validate the complete functionality of the system by simulating real-world scenarios without having 
requiring setting up cursor or claude. 

The tests will perform the following actions:

- Set up MCP Server with HTTP transport in a container. 
- Run the integration tests using the provided DigitalOcean API token and URL.

Usage:

```bash
DIGITALOCEAN_API_TOKEN=your_token_here  DIGITALOCEAN_API_URL=https://api.digitalocean.com go test -v -tags=integration ./testing/...
```

