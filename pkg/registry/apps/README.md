## App Platform MCP Tools

This directory contains tools for the App Platform MCP Server. These tools are used to interact with the App Platform API and perform various operations on App Platform resources.

## Supported Tools

- `create-app-from-spec`: This endpoint would cover initializing a new App Platform app by connecting a GitHub, GitLab, or Bitbucket repo (including specifying the branch and build settings). It condenses the app creation workflow into one action for the agent. This would let an AI assistant say “Deploy my repo X as an app” and handle the rest.
- `apps-update`: Modify an app’s settings or trigger a re-deploy. A single update-app action would let the agent change common configuration knobs without manual steps. This could include updating environment variables or secrets, scaling parameters (like instance size or count), or even changing the git branch/deploy context. It would also allow redeploying the app (e.g. if code has changed or after config updates) as part of the update. By offering an update-app endpoint, App Platform would enable flows like “the agent writes some code change to Git and then calls update-app to deploy the latest version” all in one go.
- `apps-delete`: Delete an App Platform app.
- `apps-get-info`: Get the details and status of an existing app. An agent should be able to query an app’s configuration and current state. A get-app-info endpoint would return details like the app’s name, URL, active deployment status, git source, environment variables, and health/current runtime status. This lets an AI verify what’s running – e.g. “Check if my app is deployed and what its URL is” or “What env vars does app X have?”. Keeping this read-only query separate is useful for the agent to plan next steps based on app state.
- `apps-usage`: Useful for getting live information about an app’s resource usage, like CPU and memory consumption. This could help an agent monitor app performance or diagnose issues. An agent could query this to answer questions like “How much CPU is my app using?” or “What’s the memory usage of app X?”.
- `apps-get-deployment-status`: Check the status of a specific deployment for an App Platform app. This is useful for monitoring and verifying deployments.
- `apps-list`: List all App Platform apps in the account. This allows an agent to see what apps are available and their current status.

## Example queries using App Platform MCP Tools

- Can you deploy this app from this git repository?
- Show me all of my apps in app platform.
- Delete this application for me.
- Give me the deployment status of this app.
- Which environment variables are set for this app?
- Trigger a new deployment for my app.
- Update the instance size for my app.