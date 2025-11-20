# Vercel to App Platform Migration Instructions (v3)

📦 **Version 3 Updates:**
- **NEW: Step 1.5 - Monorepo Dependency Consolidation** - Critical section on how App Platform handles monorepo dependencies
- **KEY INSIGHT:** App Platform only installs dependencies from root `package.json` in monorepos
- **PROACTIVE APPROACH:** Analyze and consolidate dependencies before deployment to avoid build failures
- **HYBRID STRATEGY:** Combine proactive analysis with fallback iterative fixing for best results

⚠️ **Important: This is a Starter Guide for AI Coding Assistants**

This guide helps AI coding assistants (Claude Code, Cursor, Gemini, etc.) collaborate with you to deploy applications to DigitalOcean App Platform.

**Constants:**
- **RETRY_LIMIT:** 3

**What this guide IS:**
- A structured workflow for AI-assisted deployment and migration
- A comprehensive checklist of steps and considerations
- Assisted error detection with fix suggestions (up to **RETRY_LIMIT** retry attempts)
- Truthful about limitations and manual steps required
- A starting point that handles common scenarios

**What this guide IS NOT:**
- A "magic migration button" that works without human oversight
- A guarantee of 1:1 feature parity with Vercel
- A replacement for understanding your application's architecture
- Suitable for highly complex, non-standard architectures without modifications
- Fully automated - you remain in control of all decisions

**Your role (the human):**
- Understand your application's architecture and requirements
- Make decisions about trade-offs and implementation approaches
- Verify deployment results and test functionality
- Implement code changes for unsupported features
- Monitor and validate production deployment

**AI assistant's role:**
- Guide you through the migration process step-by-step
- Automate repository and infrastructure setup tasks
- Detect common errors and suggest fixes
- Provide alternatives for unsupported features
- Generate configuration files and commands

**Success requires active collaboration between you and your AI assistant.**

---

## Supported Deployment Scenarios

This guide supports multiple scenarios:
1. **Migrating from existing Vercel deployment** - Moving a working Vercel app to App Platform (with or without Vercel CLI access)
2. **Deploying from GitHub repository** - App may be running on Vercel, but you just provide the repo to deploy to App Platform
3. **Deploying source code directly** - Getting source code working on App Platform (no Vercel deployment required)
4. **Deploying without repository access** - Creating new repository or deploying from local source

Follow these instructions carefully to ensure a smooth deployment.

## Initial Setup

### Determine Deployment Scenario

First, determine the user's scenario:

1. **Scenario A: Full Vercel Migration (with credentials)**
   - User has an existing live Vercel app they want to migrate
   - User has access to Vercel account/CLI
   - Ask for the Vercel app name
   - Will use Vercel CLI to pull environment variables and app configuration
   - Will migrate domains and settings

2. **Scenario B: Repository-based Deployment (app may be on Vercel)**
   - User provides a GitHub repository URL
   - App may or may not be currently running on Vercel
   - User says: "Here's my GitHub repo, deploy it to App Platform"
   - No need for Vercel CLI access - work directly from repository
   - Environment variables collected manually from .env.example or user input

3. **Scenario C: Source Code Only (no Vercel, no repo)**
   - User has source code and wants to deploy to App Platform
   - No Vercel deployment exists
   - May or may not have existing GitHub repository
   - Goal is to get the application working on App Platform

4. **Scenario D: No Repository Access**
   - User wants to deploy without giving access to personal/private repo
   - Can create new repository in their account using `gh` CLI
   - Can fork existing repository
   - Can deploy from local source (less recommended)

**For most migrations:** Scenario A or B are most common. Scenario B is simpler if user doesn't need/want to provide Vercel credentials.

### Verify Available Tools

**Primary Method: CLI Tools (Recommended)**
- **doctl CLI:** Primary method for App Platform operations (create apps, manage deployments, configure domains)
- **gh CLI:** For GitHub repository operations (create repo, fork, push code)
- **Vercel CLI (if migrating from Vercel):** For pulling environment variables and app details

**Alternative Method: MCP Servers (Optional)**
- **Vercel MCP server:** For project information (where available) - Optional
- **DigitalOcean MCP server:** For App Platform operations - Optional
- **Note:** CLI tools are preferred and don't require MCP server setup

**Verify available tools:**
- Check `doctl` is installed and authenticated: `doctl auth list`
- Check `gh` CLI is installed and authenticated: `gh auth status`
- If migrating from Vercel, check `vercel` CLI is installed: `vercel --version`

**Important Notes:**
- **CLI tools are the primary method** - MCP servers are optional
- Users don't need MCP servers - `doctl` and `gh` CLI are sufficient
- If user doesn't have a GitHub repository, we can create one using `gh` CLI
- If user doesn't want to give repo access, we can deploy from local source or create new repo

## Relevant App Platform Documentations

- [llms.txt](https://docs.digitalocean.com/products/app-platform/llms.txt)
- [nodejs specific buildpack instructions](https://docs.digitalocean.com/products/app-platform/reference/buildpacks/nodejs/)
- [App Spec reference](https://docs.digitalocean.com/products/app-platform/reference/app-spec/)

## Migration Process

### Step 0: Local Testing and Validation (Recommended)

**Before deploying to App Platform, test locally to catch issues early and avoid deployment wait time.**

1. **Verify local environment:**
   - Check Node.js version: `node --version`
   - Check package manager: `pnpm --version`, `npm --version`, or `yarn --version`
   - Verify lockfile exists: `pnpm-lock.yaml`, `package-lock.json`, or `yarn.lock`

2. **Install dependencies locally:**
   - Run: `pnpm install` (or `npm install` or `yarn install`)
   - Verify all dependencies install successfully
   - Check for any errors or warnings

3. **Test build locally:**
   - Run: `pnpm build` (or `npm run build` or `yarn build`)
   - Verify build completes successfully
   - Check for build errors or warnings
   - Verify build output directory exists (`.next` for Next.js, `dist` for others)

4. **Verify build artifacts:**
   - Check build output directory structure
   - Verify static files are generated
   - Check for API routes in build output (if applicable)
   - Verify middleware is compiled (if applicable)

5. **Check for hardcoded values:**
   - Search codebase for hardcoded URLs or Vercel-specific code
   - Verify environment variable usage
   - Check for PORT usage (should use `process.env.PORT`)

6. **Validate database migrations (if applicable):**
   - Check migration files exist
   - Verify database configuration
   - Test migration command (if database available)

### Step 1: Repository Setup

#### 1.1 Determine Repository Strategy

**Option A: Use Existing Repository (if user has one)**
- Check if current directory is a git repository: `git remote -v`
- If repository exists, verify it's accessible
- Use existing repository URL for App Platform

**Option B: Create New Repository (if user wants new repo)**
- Use `gh` CLI to create new repository: `gh repo create <repo-name> --private --source=. --push`
- Or: `gh repo create <repo-name> --public --source=. --push`
- Push current code to new repository
- Use new repository URL for App Platform

**Option C: Fork Existing Repository**
- Use `gh` CLI to fork: `gh repo fork <owner>/<repo-name> --clone=false`
- Add fork as remote: `git remote add fork https://github.com/<user>/<repo-name>.git`
- Push to fork: `git push fork main`
- Use forked repository URL for App Platform

**Option D: Deploy from Local Source (if user doesn't want repo access)**
- Can deploy directly from local source using `doctl apps create --spec app-spec.yaml`
- Note: This requires manual code updates - recommend using repository for easier updates

#### 1.2 Get App Information (If Migrating from Vercel)

**Only perform this step if user has existing Vercel deployment:**

1. **Create a local migration folder:**
   - Pattern: `./vercel-migration/<vercel-app-name>-migration/`
   - Example: `mkdir -p ./vercel-migration/my-app-migration`
   - Navigate to it: `cd ./vercel-migration/<vercel-app-name>-migration/`

2. **Link to Vercel project (if not already linked):**
   - Check if already linked: look for `.vercel` directory
   - If NOT linked, run: `vercel link --project <your-project-name> -y`
   - If already linked, skip this step to avoid double linking

3. **Pull environment variables from Vercel:**
   - Ask the user which environment to pull from: production, development, preview, or other
   - Run: `vercel env pull .env.<environment> --environment=<environment>`
   - Examples:
     - Production: `vercel env pull .env.production --environment=production`
     - Development: `vercel env pull .env.development --environment=development`
     - Preview: `vercel env pull .env.preview --environment=preview`
   - Verify the file was created: `ls -la .env.<environment>`
   - If unable to pull environment variables, leave them blank or empty when creating the App Platform app

4. **Get app information using Vercel CLI:**
   - Use `vercel inspect <app-name>` or `vercel ls` to find the app
   - Collect: App name, repository URL, deployment branch, build command, custom domains, etc.

5. **Show the user** what you found and confirm they want to proceed
   - Include the list of domains that will need DNS updates (if migrating from Vercel)
   - **Note:** Build and run commands will be validated in the next step by checking the repository

**If NOT migrating from Vercel (deploying source code directly):**
- Skip Vercel-specific steps
- Proceed directly to Step 1.5: Monorepo Dependency Consolidation (if applicable)

---

### Step 1.5: Monorepo Dependency Consolidation (CRITICAL for Monorepos)

⚠️ **IMPORTANT: App Platform Monorepo Behavior**

When deploying a monorepo with multiple components, App Platform has a specific behavior that you MUST understand:

**Key Insight:**
- App Platform installs dependencies **only from the root `package.json`**
- Each component's `source_dir` only affects where build/run commands execute
- Dependencies in component-specific `package.json` files are NOT automatically installed
- This means build tools like `tsx`, `typescript`, `pnpm`, etc. must be in root `package.json`

**Why This Matters:**

If your monorepo structure looks like this:
```
monorepo/
├── package.json          # Root - App Platform installs dependencies from HERE
├── apps/
│   ├── dashboard/
│   │   └── package.json  # Component - dependencies NOT installed automatically
│   └── api/
│       └── package.json  # Component - dependencies NOT installed automatically
```

And `apps/api/package.json` has `tsx` as a dependency, but root `package.json` doesn't, the API build will fail with "command not found: tsx".

---

#### 1.5.1: Analyze Monorepo Dependencies (Proactive Approach - Recommended)

**Before deploying**, analyze all component package.json files and consolidate dependencies:

1. **Detect if this is a monorepo:**
   - Check for `pnpm-workspace.yaml`, `lerna.json`, `nx.json`
   - Check if root `package.json` has "workspaces" field
   - Look for multiple `package.json` files in subdirectories

2. **If monorepo detected, analyze component dependencies:**

   ```bash
   # Read all component package.json files
   # Extract dependencies and devDependencies
   # Identify build-time tools needed
   ```

3. **Categorize required packages:**

   **Build-Time Tools (must be in root devDependencies):**
   - `tsx` - TypeScript execution for Node.js
   - `typescript` - TypeScript compiler
   - `@types/node` - Node.js type definitions
   - `@types/*` - Any TypeScript type packages
   - Build tools: `vite`, `webpack`, `rollup`, `esbuild`
   - Linters: `eslint`, `prettier`
   - Testing: `jest`, `vitest`, `playwright`

   **Runtime Dependencies (must be in root dependencies):**
   - Framework packages: `next`, `react`, `express`, `vue`
   - Runtime utilities needed by any component
   - Shared packages used across components

4. **Propose root package.json updates to user:**

   ```
   I detected this is a monorepo with 2 services. App Platform only installs
   dependencies from the root package.json.

   I need to add these build-time tools to root package.json:
   - tsx (for API TypeScript execution)
   - typescript (for type checking)
   - @types/node (for Node types)

   Would you like me to:
   A) Add these automatically and proceed
   B) Skip this and let build fail (then fix iteratively)
   C) Review the full dependency list first
   ```

5. **If user approves, update root package.json:**

   ```bash
   # Merge component dependencies into root package.json
   # Keep existing root dependencies
   # Add missing build tools to devDependencies
   # Add missing runtime deps to dependencies
   ```

**Example Transformation:**

**Before (will fail on App Platform):**
```json
// Root package.json
{
  "name": "my-monorepo",
  "private": true,
  "devDependencies": {
    "turbo": "^2.5.5"
  }
}

// apps/api/package.json
{
  "name": "api",
  "dependencies": {
    "express": "^4.18.2"
  },
  "devDependencies": {
    "tsx": "^4.20.6",
    "typescript": "^5.0.0"
  }
}
```

**After (will work on App Platform):**
```json
// Root package.json
{
  "name": "my-monorepo",
  "private": true,
  "dependencies": {
    "express": "^4.18.2"  // Added from api
  },
  "devDependencies": {
    "turbo": "^2.5.5",
    "tsx": "^4.20.6",      // Added from api
    "typescript": "^5.0.0",  // Added from api
    "@types/node": "^20.0.0"  // Added from api
  }
}
```

---

#### 1.5.2: Iterative Dependency Fixing (Fallback Approach)

If you skip proactive analysis or it misses something:

1. **Let the build attempt**
2. **Parse build error logs for missing dependencies:**

   Common error patterns:
   ```
   /bin/sh: tsx: not found
   /bin/sh: tsc: not found
   Error: Cannot find module 'typescript'
   ```

3. **Suggest fix to user:**

   ```
   Build failed because 'tsx' is not found.

   This is a monorepo and App Platform only installs dependencies from
   root package.json.

   Fix: Add 'tsx' to root package.json devDependencies

   Would you like me to add it and redeploy?
   ```

4. **Update root package.json and retry:**
   - Add missing package to appropriate section
   - Commit and push changes
   - Redeploy (automatic if `deploy_on_push: true`)

5. **Repeat up to 3 times** for different missing packages

---

#### 1.5.3: Trade-offs and Considerations

**Approach A: Consolidate All Dependencies (Proactive)**

**Pros:**
- ✅ Works on first deployment
- ✅ No iteration needed
- ✅ Predictable behavior
- ✅ User stays in control (approves changes)

**Cons:**
- ❌ All components get ALL dependencies
- ❌ Larger `node_modules` in each component
- ❌ Slower installs
- ❌ Slightly larger deployment artifacts

**When to use:**
- Small monorepos (2-4 services)
- "Just make it work" scenarios
- User wants fast deployment

**Approach B: Iterative Fixing (Reactive)**

**Pros:**
- ✅ Only adds truly needed packages
- ✅ Cleaner dependency tree
- ✅ Educational for user

**Cons:**
- ❌ Requires multiple deploy attempts (2-5 minutes each)
- ❌ More complex error detection
- ❌ Could miss cascading errors

**When to use:**
- Large monorepos with many services
- Production apps where bundle size matters
- User wants optimal configuration

**Approach C: Hybrid (Recommended)**

1. Proactively analyze and propose changes
2. User reviews and approves
3. If build still fails, fall back to iterative fixing
4. Usually works first time, graceful degradation

---

#### 1.5.4: Special Cases

**Case 1: Using pnpm workspaces**

If `pnpm-workspace.yaml` exists:
```yaml
packages:
  - 'apps/*'
  - 'packages/*'
```

App Platform buildpack automatically:
- Detects pnpm from `pnpm-lock.yaml`
- Runs `pnpm install` at root
- Respects workspace configuration

**Still need**: Build tools in root package.json

**Case 2: Using npm/yarn workspaces**

If root `package.json` has:
```json
{
  "workspaces": ["apps/*", "packages/*"]
}
```

App Platform buildpack automatically:
- Detects workspaces
- Runs `npm install` or `yarn install` at root
- Links workspace packages

**Still need**: Build tools in root package.json

**Case 3: Components use different runtimes**

If dashboard uses `pnpm build` but API uses `tsx src/index.ts`:
- Both commands must have their dependencies in root
- `pnpm` must be in root (detected from lockfile)
- `tsx` must be in root devDependencies

---

#### 1.5.5: Implementation Example (AI Assistant Instructions)

**When detecting a monorepo:**

```typescript
// Pseudo-code for AI assistant

function analyzeMonorepo(repoPath) {
  // 1. Detect monorepo structure
  const isMonorepo = hasWorkspaceConfig(repoPath);
  if (!isMonorepo) return null;

  // 2. Find all component package.json files
  const components = findComponents(repoPath);

  // 3. Extract dependencies from each component
  const allDeps = {
    dependencies: new Set(),
    devDependencies: new Set()
  };

  for (const component of components) {
    const pkg = readPackageJson(component.path);

    // Merge runtime dependencies
    Object.keys(pkg.dependencies || {}).forEach(dep =>
      allDeps.dependencies.add({ name: dep, version: pkg.dependencies[dep] })
    );

    // Merge build-time dependencies
    Object.keys(pkg.devDependencies || {}).forEach(dep =>
      allDeps.devDependencies.add({ name: dep, version: pkg.devDependencies[dep] })
    );
  }

  // 4. Check what's missing from root
  const rootPkg = readPackageJson(`${repoPath}/package.json`);
  const missing = {
    dependencies: [],
    devDependencies: []
  };

  allDeps.dependencies.forEach(dep => {
    if (!rootPkg.dependencies?.[dep.name]) {
      missing.dependencies.push(dep);
    }
  });

  allDeps.devDependencies.forEach(dep => {
    if (!rootPkg.devDependencies?.[dep.name]) {
      missing.devDependencies.push(dep);
    }
  });

  // 5. Return analysis
  return {
    isMonorepo: true,
    components: components.length,
    missing,
    recommendation: missing.dependencies.length + missing.devDependencies.length > 0
      ? "CONSOLIDATE_DEPENDENCIES"
      : "NO_ACTION_NEEDED"
  };
}

// Then ask user
if (analysis.recommendation === "CONSOLIDATE_DEPENDENCIES") {
  askUser(`
    Detected monorepo with ${analysis.components} components.

    App Platform only installs dependencies from root package.json.

    Missing from root:
    ${formatDependencies(analysis.missing)}

    Add these to root package.json?
    A) Yes, add automatically
    B) No, I'll handle it manually
    C) Show me the full proposed package.json
  `);
}
```

---

#### 1.5.6: Troubleshooting Monorepo Dependency Issues

**Symptom:** Build fails with "command not found: tsx"

**Diagnosis:**
```bash
# Check build logs
doctl apps logs <app-id> --type build

# Look for:
/bin/sh: tsx: not found
/bin/sh: 1: tsx: not found
```

**Root Cause:** `tsx` is in component `package.json` but not in root

**Fix:**
```bash
# Add to root package.json devDependencies
{
  "devDependencies": {
    "tsx": "^4.20.6"
  }
}

# Commit and push
git add package.json
git commit -m "Add tsx to root for App Platform build"
git push
```

**Symptom:** Build succeeds but run fails with "Cannot find module 'express'"

**Diagnosis:**
```bash
# Check runtime logs
doctl apps logs <app-id> --type run

# Look for:
Error: Cannot find module 'express'
```

**Root Cause:** `express` is in component `package.json` but not in root

**Fix:**
```bash
# Add to root package.json dependencies (NOT devDependencies)
{
  "dependencies": {
    "express": "^4.18.2"
  }
}
```

---

#### 1.5.7: Validation Checklist

Before proceeding to deployment:

- [ ] Identified if repository is a monorepo
- [ ] If monorepo: Analyzed all component package.json files
- [ ] Identified build-time tools needed (tsx, typescript, etc.)
- [ ] Identified runtime dependencies needed
- [ ] Updated root package.json with consolidated dependencies
- [ ] User approved dependency consolidation
- [ ] Committed changes to git
- [ ] Ready to proceed to app spec creation

---

### Step 2: App Migration

Follow these steps to migrate the app:

#### 2.1 Project settings and Build Configuration

**Inspect the repository to understand project structure and let App Platform buildpack auto-detect build/run commands**

1. **Fetch and inspect package.json from the repository:**
   - If monorepo or source_dir is specified: look at `[source_dir]/package.json`. Deploy them as separate components per app.
   - If root level app: look at root `package.json`
   - Identify the framework from dependencies (Next.js, React, Vue, Express, etc.)
   - Note if there's a "build" script and "start" script in the scripts section
   - Check the "engines" field for Node.js version requirements

2. **Check Vercel configuration for custom settings:**
   - Look in `vercel.json` for `buildCommand`, `outputDirectory`, or `framework` settings
   - Check Vercel project settings for any custom build/install commands
   - Note the framework detected by Vercel
   - If Vercel uses custom commands, consider using them; otherwise leave empty for buildpack auto-detection

3. **Validation before migration:**

   **Project Structure:**
   - Verify package.json exists at the correct location (root or source_dir)
   - Check for package-lock.json or yarn.lock (for dependency consistency)
   - For monorepos: verify source_dir path is correct relative to repository root
   - Supported Node.js version in "engines" field in App Platform, (from https://docs.digitalocean.com/products/app-platform/reference/buildpacks/nodejs/#current-buildpack-version-and-supported-runtimes)

   **Build/Run Commands - Prefer buildpack auto-detection:**
   - **Default approach:** Leave build and run commands empty to let App Platform buildpack auto-detect based on framework
   - **Only specify custom commands if:**
     - Vercel uses non-standard build commands (check vercel.json)
     - The project has custom scripts that differ from framework defaults
     - You need to override buildpack defaults for specific reasons
   - **Reference:** [App Platform Node.js buildpack documentation](https://docs.digitalocean.com/products/app-platform/reference/buildpacks/nodejs/) for auto-detection behavior

   **For Static Sites:**
   - Buildpack will auto-detect framework and build command
   - Specify `output_dir` only if different from framework default (e.g., Next.js uses `out` for static export, not `.next`)
   - Common output directories: `build`, `dist`, `out`, `public`, `_site`

   **For Web Services:**
   - Buildpack will auto-detect framework and start command
   - Ensure app listens on `process.env.PORT` (App Platform injects this variable)
   - Leave run command empty unless using a custom start script

   **Install Command:**
   - Buildpack auto-detects npm, yarn, or pnpm based on lockfile
   - Only specify custom install command if needed (e.g., monorepo workspaces)

   **If you need to specify commands:**
   - Verify the script exists in package.json "scripts" section
   - Or verify the framework package is in dependencies/devDependencies
   - Ask the user to confirm if uncertain

#### 2.2 Environment Variables

1. **Collect environment variables:**

   **If migrating from Vercel:**
   - Read the `.env.<environment>` file from the migration folder (Step 1.1)
   - Parse all variables from this file
   - If no environment variables were pulled, proceed with manual collection

   **If deploying source code directly (no Vercel):**
   - Check for `.env.example` or `.env.sample` file in project
   - Read `.env.example` to understand required variables
   - Ask user for values for each required variable
   - Generate secure values where needed (e.g., `AUTH_SECRET`)

2. **Transform system variables:**
   - Note: `VERCEL_URL`, `VERCEL_ENV`, any `VERCEL_GIT_*` and Vercel-specific system variables will NOT be available in App Platform
   - Warn the user if the app depends on these variables
   - Replace `VERCEL_URL` with App Platform URL (set after deployment)
   - Replace `VERCEL_ENV` with `NODE_ENV=production`

3. **Prepare environment variables for App Platform:**
   - Keep the same variable names (except for Vercel-specific ones)
   - Mark sensitive variables (API keys, secrets, database passwords) as `type: SECRET` in app spec
   - Generate secure random values where needed:
     - `AUTH_SECRET`: `openssl rand -base64 32`
     - `STRIPE_WEBHOOK_SECRET`: User provides or placeholder
   - If unable to retrieve values, ask user or leave blank (app may have defaults)

4. **Database connection string:**
   - If creating managed database in App Platform, use `${db.DATABASE_URL}` reference
   - If using external database, ask user for connection string

#### 2.3 Component Type Detection

Determine the appropriate App Platform component type based on the app's characteristics:

**Static Sites:**
- Pure static HTML/CSS/JS with no server-side logic
- Static site generators (Gatsby, Hugo, Jekyll, etc.) that build to an output directory
- React/Vue/Angular SPAs that are fully client-side rendered
- **Create as Static Site component**
- **Configuration needed:**
  - `build_command`: Leave empty for buildpack auto-detection (or specify if custom)
  - `output_dir`: Only if different from framework default (e.g., `build`, `dist`, `out`, `public`)

**Web Services (Dynamic Sites):**
- Next.js apps with API routes (files in `/pages/api/*` or `/app/api/*`)
- Next.js apps using server-side rendering (SSR) or server components
- Node.js/Express backend applications
- Any app that needs a running server process
- Apps with serverless functions in `/api/*` directory
- **Create as Web Service component**
- **Configuration needed:**
  - `build_command`: Leave empty for buildpack auto-detection (or specify if custom)
  - `run_command`: Leave empty for buildpack auto-detection (or specify if custom)
  - No `output_dir` - the service runs continuously

**How to determine:**
1. Check if app has `/api/*` routes or serverless functions → **Web Service**
2. Check if Next.js uses `getServerSideProps`, `getInitialProps`, or server components → **Web Service**
3. Check if there's a server file (like `server.js`, `index.js` with Express) → **Web Service**
4. If purely static output with no server logic → **Static Site**
5. When in doubt, check the Vercel deployment type or framework detection

**Multiple Components (Advanced):**
- If the app has both static content AND API routes, you could separate into:
  - Option A: Single web service (simplest - handles both static and API)
  - Option B: Separate static site + web service components (more complex setup)
- **Recommend Option A** unless user specifically wants separation
- **See section 2.3.5 below for detailed multi-service architecture guidance**

#### 2.3.5 Multi-Service and Monorepo Applications

**App Platform supports deploying multiple components in a single app.** This is different from deploying multiple separate apps.

**One App with Multiple Components vs Multiple Apps:**

| Use One App (Multiple Components) | Use Multiple Apps |
|-----------------------------------|-------------------|
| Services in same repository | Services in different repositories |
| Deploy together on git push | Deploy independently |
| Shared database connection | Separate databases |
| Internal communication (fast, no public internet) | Public API communication |
| Examples: monorepo, microservices, frontend+API+worker | Examples: different products, different teams |

**Component Types in One App:**
- **Multiple web services** (e.g., frontend, API, admin panel)
- **Static sites + web services** (e.g., marketing site + API)
- **Workers** (background jobs, queue processors, continuous processes)
- **Jobs** (one-time tasks, pre-deploy hooks, post-deploy hooks)
- **Databases** (PostgreSQL, ValKey/Redis)

---

**Example 1: Monorepo with Frontend + API + Shared Database**

This is the most common multi-service scenario.

**Repository structure:**
```
monorepo/
├── apps/
│   ├── web/          # Next.js frontend
│   │   ├── package.json
│   │   └── src/
│   └── api/          # Express API
│       ├── package.json
│       └── src/
├── packages/
│   └── shared/       # Shared code
│       └── package.json
├── package.json      # Root workspace config
└── pnpm-workspace.yaml  # or package-lock.json, yarn.lock
```

**app-spec.yaml:**
```yaml
name: my-fullstack-app
region: nyc1

services:
  # Frontend service
  - name: web
    github:
      repo: owner/monorepo
      branch: main
      deploy_on_push: true
    source_dir: /apps/web
    build_command: ""  # Auto-detect Next.js
    run_command: ""    # Auto-detect Next.js start
    http_port: 8080
    instance_count: 1
    instance_size_slug: basic-xxs
    routes:
      - path: /       # Frontend handles root and all non-API routes
    envs:
      - key: API_URL
        value: ${api.PRIVATE_URL}  # Internal URL to API service (fast, no public internet)
      - key: DATABASE_URL
        value: ${db.DATABASE_URL}
        scope: RUN_AND_BUILD_TIME
      - key: NODE_ENV
        value: production

  # API service
  - name: api
    github:
      repo: owner/monorepo
      branch: main
      deploy_on_push: true
    source_dir: /apps/api
    build_command: ""  # Auto-detect or specify if custom
    run_command: ""    # Auto-detect Express/Node
    http_port: 8080
    instance_count: 1
    instance_size_slug: basic-xxs
    routes:
      - path: /api    # API handles all /api/* routes
    envs:
      - key: DATABASE_URL
        value: ${db.DATABASE_URL}
        scope: RUN_AND_BUILD_TIME
      - key: NODE_ENV
        value: production

databases:
  - name: db
    engine: PG
    version: "18"
    production: false  # Set to true for production tier with backups
```

**Key Points:**
1. Both services reference same repository, different `source_dir`
2. Services can reference each other with `${service-name.PRIVATE_URL}` (internal communication, fast)
3. Public routes are configured per service (`/` for web, `/api` for API)
4. Database is shared via `${db.DATABASE_URL}` reference
5. All services deploy together when code is pushed
6. Buildpack auto-detects framework in each service's directory

**Monorepo Package Manager Detection:**
- If `pnpm-workspace.yaml` exists → buildpack uses pnpm
- If root `package.json` has "workspaces" field → buildpack uses npm/yarn workspaces
- Buildpack automatically installs dependencies for each service

---

**Example 2: Adding Workers and Jobs for Background Tasks**

**Repository structure:**
```
app/
├── web/              # Frontend
├── api/              # API
├── worker/           # Background worker (continuous)
│   └── worker.js     # Processes queue, runs continuously
└── jobs/
    └── migrations/   # Database migrations (one-time)
        └── migrate.js
```

**app-spec.yaml additions:**
```yaml
name: my-app-with-workers
region: nyc1

services:
  # ... web and api services as above ...

workers:
  # Background worker - runs continuously
  - name: email-worker
    github:
      repo: owner/repo
      branch: main
    source_dir: /worker
    build_command: ""
    run_command: "node worker.js"  # Continuous process
    instance_count: 1
    instance_size_slug: basic-xxs
    envs:
      - key: DATABASE_URL
        value: ${db.DATABASE_URL}
      - key: REDIS_URL
        value: ${cache.DATABASE_URL}
      - key: NODE_ENV
        value: production

jobs:
  # Pre-deploy job - runs before deployment
  - name: db-migrate
    kind: PRE_DEPLOY
    github:
      repo: owner/repo
      branch: main
    source_dir: /jobs/migrations
    build_command: ""
    run_command: "node migrate.js"  # Runs once before deployment
    instance_size_slug: basic-xxs
    envs:
      - key: DATABASE_URL
        value: ${db.DATABASE_URL}

databases:
  - name: db
    engine: PG
    version: "18"
  - name: cache
    engine: REDIS  # ValKey (Redis fork)
    version: "7"
```

**Worker vs Job:**
- **Worker**: Continuous process (e.g., queue processor, scheduled task runner)
- **Job**: Runs once per deployment (e.g., database migrations, cache warming)
- **Job kinds**:
  - `PRE_DEPLOY`: Runs before new code is deployed (migrations)
  - `POST_DEPLOY`: Runs after deployment succeeds (cache warming)

---

**Example 3: Separating Static Site from API Service**

Sometimes you want a separate static frontend (SPA) and API backend.

**app-spec.yaml:**
```yaml
name: spa-with-api
region: nyc1

static_sites:
  # Static React/Vue/Angular SPA
  - name: web
    github:
      repo: owner/repo
      branch: main
    source_dir: /frontend
    build_command: "npm run build"  # Or leave empty for auto-detect
    output_dir: dist  # Or build, out, public
    routes:
      - path: /
    envs:
      - key: VITE_API_URL  # Or REACT_APP_API_URL, NEXT_PUBLIC_API_URL
        value: ${api.PUBLIC_URL}  # Public URL to API

services:
  # API service
  - name: api
    github:
      repo: owner/repo
      branch: main
    source_dir: /backend
    build_command: ""
    run_command: ""
    http_port: 8080
    routes:
      - path: /api
    envs:
      - key: DATABASE_URL
        value: ${db.DATABASE_URL}

databases:
  - name: db
    engine: PG
    version: "18"
```

**When to use this pattern:**
- SPA that's 100% client-side rendered
- Clear separation between frontend and backend
- Frontend only needs static file hosting

---

**When to Use Multiple Apps vs One App with Multiple Components:**

**Use ONE app with multiple components when:**
- ✅ All services are in the same repository (monorepo)
- ✅ Services share the same database
- ✅ Services need to deploy together (atomic deployments)
- ✅ Services communicate internally (low latency requirements)
- ✅ Simpler management (one dashboard, one app spec)

**Use MULTIPLE separate apps when:**
- ✅ Services are in different repositories
- ✅ Services need to deploy independently
- ✅ Services have different teams/ownership
- ✅ Services need independent scaling and resource management
- ✅ Services are truly separate products/systems

---

**Deployment Behavior with Multiple Components:**
- Git push triggers deployment of **ALL components** in the app
- All components must build successfully before deployment
- If **one component fails**, entire deployment fails (rollback to previous)
- Components start simultaneously after successful build
- Database migrations (PRE_DEPLOY jobs) run before services start

**Troubleshooting Multi-Component Deployments:**

If one component fails:
1. Check which component failed in logs: `doctl apps logs <app-id> --type build`
2. Fix the failing component's code or configuration
3. Push fix - all components rebuild
4. Consider deploying components in separate apps if you need independent deployments

**Internal Service Communication:**
- Use `${service-name.PRIVATE_URL}` for fast internal communication (no public internet)
- Example: Frontend calls `${api.PRIVATE_URL}/users` instead of public URL
- Faster and more secure than public URLs
- Private URLs are only accessible within the app's components

**Assistant Instructions:**
- When detecting a monorepo, ask user: "I see this is a monorepo with multiple services. Would you like to deploy them as separate components in one app, or as a single service?"
- Show the user which services you detected (by finding multiple package.json files)
- Propose an app spec with multiple components
- Explain that all services will deploy together
- Confirm the routing strategy (which paths go to which services)

#### 2.4 Feature Translation

**Serverless Functions → Web Services or Functions:**
- Vercel serverless functions in `/api/*` directory
- Migrate to App Platform web service with API endpoints
- OR guide user to use App Platform Functions
- Note: Edge Functions are NOT directly supported - must refactor to web services

**Cron Jobs:**
- If Vercel has cron jobs invoking API endpoints
- Create native App Platform cron jobs
- Map cron expressions directly
- Update to use App Platform internal URLs instead of `/api/*` HTTP endpoints

**Middleware:**
- Vercel `middleware.ts` is NOT supported in App Platform
- Inform user they need to implement application-level middleware
- This requires code changes in the application itself

**ISR (Incremental Static Regeneration):**
- NOT directly supported in App Platform
- Inform user they have these options:
  - Rebuild static site periodically using cron jobs
  - Convert to server-side rendering (SSR)
  - Use Spaces with cache persistence for future NFS support

**Image Optimization:**
- NOT natively supported in App Platform
- Recommend these alternatives:
  1. Cloudflare Images API (fully managed, paid after 5000 transformations)
  2. Next.js `next/image` with Sharp in the same container
  3. Dedicated worker or cron job with Sharp/libvips processing to Spaces
  4. External image CDN service

**Headers and Redirects:**
- Vercel `headers` in `vercel.json` → Configure CORS in app spec or application middleware
- Vercel `redirects` in `vercel.json` → App Platform routing rules in app spec
- Vercel `rewrites` in `vercel.json` → Internal routing configuration in app spec

**Edge Caching:**
- Vercel has configurable multi-tier caching
- App Platform has standard CDN caching (cannot disable or configure extensively)
- Guide user to adjust cache headers in their application

#### 2.5 Create App Platform App

**Using doctl CLI (Primary Method):**

1. **Create app spec YAML file** with configuration based on component type:

   **IMPORTANT: Only proceed with app creation after completing validation in step 2.1 and local testing in Step 0**

   **For Static Sites:**
   ```yaml
   name: <app-name>
   region: <region>  # e.g., nyc1, sfo1, ams3
   services:
     - name: web
       github:
         repo: <owner>/<repo-name>
         branch: main  # or from Vercel if migrating
         deploy_on_push: true
       source_dir: /  # or path for monorepo
       build_command: ""  # empty for buildpack auto-detection
       output_dir: ""  # only if non-standard
       instance_count: 1
       instance_size_slug: basic-xxs
       envs:
         # Add environment variables here
   ```

   **For Web Services (Dynamic Sites):**
   ```yaml
   name: <app-name>
   region: <region>  # e.g., nyc1, sfo1, ams3
   services:
     - name: web
       github:
         repo: <owner>/<repo-name>
         branch: main  # or from Vercel if migrating
         deploy_on_push: true
       source_dir: /  # or path for monorepo
       build_command: ""  # empty for buildpack auto-detection
       run_command: ""  # empty for buildpack auto-detection
       http_port: 8080
       instance_count: 1
       instance_size_slug: basic-xxs
       envs:
         - key: NODE_ENV
           value: production
           scope: RUN_AND_BUILD_TIME
         # Add other environment variables here
       databases:
         - name: db
           engine: PG
           version: "18"
           production: false
   ```

2. **Save the app spec to local filesystem:**
   - Save as `app-spec.yaml` in project root or migration folder
   - This allows the user to manually create the app if needed

3. **Create app using doctl CLI:**
   - Run: `doctl apps create --spec app-spec.yaml`
   - Capture app ID from output
   - Monitor deployment: `doctl apps get <app-id>`

4. **Set up any required databases:**
   - If app uses Vercel Postgres → Create Managed PostgreSQL
   - If app uses Vercel KV → Create Managed ValKey (Redis fork)
   - Update connection strings in environment variables

**Database Configuration Details:**

**Adding Managed Databases in app spec:**

```yaml
databases:
  # PostgreSQL database
  - name: db
    engine: PG
    version: "18"  # Or "17", "16", "15", "14"
    production: false  # false = dev tier ($15/mo), true = production tier ($25/mo+)
    cluster_name: my-db-cluster  # optional - creates new cluster or references existing

  # Redis/ValKey cache
  - name: cache
    engine: REDIS  # ValKey (Redis fork, Redis 7.2 compatible)
    version: "7"
    production: false
```

**Referencing databases in services:**

```yaml
services:
  - name: web
    # ... other config ...
    envs:
      # Database URL is injected automatically
      - key: DATABASE_URL
        value: ${db.DATABASE_URL}  # Format: postgresql://user:pass@host:port/dbname?sslmode=require
        scope: RUN_AND_BUILD_TIME  # Available during build and runtime

      # Redis URL
      - key: REDIS_URL
        value: ${cache.DATABASE_URL}  # Format: redis://user:pass@host:port
        scope: RUN_TIME  # Only available at runtime
```

**Connection string formats:**
- **PostgreSQL**: `postgresql://username:password@host:port/database?sslmode=require`
- **Redis/ValKey**: `redis://username:password@host:port`
- SSL/TLS is enforced for all connections to managed databases and cannot be disabled
- Credentials are automatically generated and rotated

**Database tiers:**
- **Development** (`production: false`): $15/month, 1 GB RAM, 10 GB disk, 1 node, daily backups
- **Production** (`production: true`): Starting at $25/month, 2 GB RAM, 25 GB disk, automatic failover, point-in-time recovery

**Using external database (not managed by App Platform):**

If you want to use an existing external database:

```yaml
services:
  - name: web
    envs:
      - key: DATABASE_URL
        value: "postgresql://external-host.com:5432/mydb?sslmode=require"
        type: SECRET  # Mark as secret to encrypt in dashboard
```

**Database migrations:**

Use PRE_DEPLOY jobs to run migrations before deployment:

```yaml
jobs:
  - name: db-migrate
    kind: PRE_DEPLOY
    github:
      repo: owner/repo
      branch: main
    source_dir: /
    run_command: "npm run migrate"  # or "pnpm migrate", "yarn migrate"
    instance_size_slug: basic-xxs
    envs:
      - key: DATABASE_URL
        value: ${db.DATABASE_URL}
```

Common migration commands:
- Prisma: `npx prisma migrate deploy`
- Drizzle: `npm run db:push` or `npm run db:migrate`
- TypeORM: `npm run typeorm migration:run`
- Knex: `npm run knex migrate:latest`
- Sequelize: `npx sequelize-cli db:migrate`

**Important notes:**
- Database creation can take 10-15 minutes on first deployment
- Databases persist across deployments and app deletions (must manually delete)
- Connection pooling is recommended (use connection pool library)
- **Data migration** from Vercel Postgres is a separate manual process (not automated by this guide)

**Data migration strategy (manual process):**
1. Create new App Platform database
2. Use `pg_dump` to export from Vercel Postgres
3. Use `psql` or `pg_restore` to import to App Platform database
4. Test thoroughly before switching DNS
5. Keep Vercel database running during transition
6. Consider using database migration tools like `pg_dump`/`pg_restore`, or managed migration services

#### 2.6 Configure Domains

**Using doctl CLI (Primary Method):**

1. **Add custom domains to the App Platform app:**
   - Run: `doctl apps create-domain <app-id> --domain <domain-name>`
   - For each domain that was in Vercel (if migrating), add it to App Platform
   - App Platform will provide DNS configuration details
   - Both apex domains (example.com) and subdomains (www.example.com) are supported
   - App Platform will automatically provision SSL certificates

2. **Get DNS configuration:**
   - Get default ingress: `doctl apps get <app-id> --format DefaultIngress`
   - Get domain configuration: `doctl apps get-domain <app-id> <domain-name>`

3. **Provide DNS update instructions to the user:**
   - Display these instructions as post deployment instruction
   - List each domain that needs to be updated
   - Ask if the user wants to manage domains through DigitalOcean or keep it self managed (outside of DigitalOcean)
   - If the user wants to manage domains through DigitalOcean, then display:
   
   ```md
   ## Move DNS records to DigitalOcean
   
   Update your DNS provider with our nameservers.
   
   DigitalOcean Nameservers
   - ns1.digitalocean.com
   - ns2.digitalocean.com
   - ns3.digitalocean.com
   ```
   
   - If the user wants to keep it self managed:
     - Get the default_ingress using `doctl apps get <app-id> --format DefaultIngress`
     - Display:
   
   ```md
   Add a new CNAME record using the provided alias to point your domain to your DigitalOcean app.
   
   ## CNAME alias
   <default_ingress>
   
   Check if CNAME flattening is supported for root domains. If your provider doesn't support this method, you can create an A record that points to your root domain.
   
   ## A Records
   - 162.159.140.98
   - 172.66.0.96
   ```
   
   - Note: default ingress is only available after a successful deployment

4. **Important notes for the user:**
   - DNS propagation can take up to 48 hours
   - Wildcard domains (*.example.com) are also supported if needed

#### 2.8 Assisted Deployment Troubleshooting

**If deployment fails, follow this assisted troubleshooting process:**

The AI assistant will analyze errors and suggest fixes for you to review and approve. This is not fully automatic - you remain in control.

1. **Check deployment status:**
   - Use `doctl apps get <app-id>` to get deployment status
   - Status will be either: ACTIVE, DEPLOYED (success) or FAILED, ERROR (failure)

2. **Fetch deployment logs immediately:**
   - Get build logs: `doctl apps logs <app-id> --type build`
   - Get runtime logs: `doctl apps logs <app-id> --type run`
   - Get deployment logs: `doctl apps get-deployment <app-id> <deployment-id>`
   - Parse the error messages to identify the failure type

3. **Analyze the error and suggest fixes:**

   **For "command not found" or "script not found" errors:**
   - **Suggested fix - prefer buildpack auto-detection:**
     - **First attempt:** Leave build/run commands empty to let buildpack auto-detect
     - This works for most standard frameworks (Next.js, React, Vue, etc.)
     - If this fails, then try checking package.json for custom scripts:
       - Read package.json from the migration folder at `./vercel-migration/<app-name>-migration/package.json`
       - If Vercel used a custom script (check vercel.json), use that specific script
       - If project has custom scripts that differ from framework defaults, specify those
   - Update the app with corrected configuration and redeploy

   **For "no start command" or run command errors:**
   - **Suggested fix - prefer buildpack auto-detection:**
     - **First attempt:** Leave run command empty to let buildpack auto-detect
     - Buildpack will use framework defaults or package.json "start" script
     - Only specify custom run command if buildpack auto-detection fails
   - Update run command and redeploy

   **For "output directory not found" errors:**
   - **Suggested fix:**
     - Check common output directories: `build`, `dist`, `out`, `public`, `.next`
     - **Check .gitignore in migration folder** for build output patterns
     - Look in build logs for "writing output to..." messages
     - Update output_dir and redeploy

   **For "module not found" or dependency errors:**
   - **Check migration folder for lockfile** (package-lock.json or yarn.lock)
   - **Suggested fix options:**
     - If lockfile is in a subdirectory, update source_dir
     - If using private packages, check if registry auth env vars are needed
     - Add NODE_ENV=production if missing

   **For port/health check failures (service starts but fails health check):**
   - Fetch runtime logs to see what port the app is listening on
   - **Suggested fix:**
     - Check if app is hardcoded to wrong port
     - Ensure PORT environment variable is set
     - Update HTTP port configuration if needed

   **For monorepo/source directory errors:**
   - **Suggested fix:**
     - Check Vercel config for rootDirectory
     - **Search migration folder** for package.json in subdirectories
     - Look for workspace configuration files (lerna.json, nx.json, etc.)
     - Update source_dir to correct path and redeploy
     - Deploy the mono repo under 1 app, where each repository service is a `service component` or `static site component` depending on the repository service. Ask the user for confirmation.

4. **Apply the fix and redeploy:**
   - Update app spec YAML with fixes
   - Update app using doctl: `doctl apps update <app-id> --spec app-spec.yaml`
   - Or update specific config: `doctl apps update <app-id> --env <key>=<value>`
   - Wait for deployment to complete and check status again

5. **Retry logic:**
   - If first suggested fix doesn't work, analyze logs again and try alternative approach
   - Maximum 3 assisted troubleshooting attempts per app
   - After 3 failed attempts, report to user with detailed error analysis and request manual intervention

6. **Report fix applied:**
   - Tell the user what error was found
   - Tell the user what fix was applied
   - Show the updated configuration

### Step 3: Validation and Automatic Recovery

After migrating the app:

1. **Verify deployment succeeded** in App Platform (use `doctl apps get <app-id>` to check status)
   - Wait for the deployment to complete (this may take several minutes)
   - Check deployment status: should be "ACTIVE" or "DEPLOYED"

2. **If deployment FAILED, begin assisted troubleshooting:**
   - Follow the process in section 2.8 "Assisted Deployment Troubleshooting"
   - Fetch logs, analyze errors, suggest fixes for user to review
   - Retry up to 3 times with different suggested approaches
   - **Continue with validation after:**
     - The app is successfully deployed, OR
     - 3 troubleshooting attempts have been made and user intervention is needed

3. **If suggested fixes don't resolve the issue after 3 attempts:**
   - Present detailed error information to the user
   - Show what fixes were attempted
   - Ask user for manual configuration input:
     - Correct build command
     - Correct output directory
     - Missing environment variables
     - Source directory path
   - Apply user's corrections and redeploy

4. **Test the deployed application:**
   - Visit the App Platform URL
   - Test critical functionality
   - Test API endpoints if present
   - Check that environment variables are working

5. **Report status to the user:**
   - Indicate the migration result: Success, Auto-Fixed, or Failed
   - If auto-fixed, show what was corrected
   - If failed, show the errors and fixes that were attempted

6. **Provide rollback plan:**
   - Inform user how to rollback if issues arise
   - See section 3.1 below for detailed rollback instructions

#### 3.1 Rollback and Recovery Plan

**Before updating DNS (if migrating from Vercel):**

1. ✅ Test App Platform deployment thoroughly using the default `.ondigitalocean.app` URL
2. ✅ Verify all functionality works (API endpoints, database, authentication, etc.)
3. ✅ Keep Vercel app running during DNS propagation (24-48 hours)
4. ✅ Monitor App Platform logs and metrics for errors
5. ✅ Have rollback plan ready before updating DNS

---

**Rollback Option A: DNS Rollback (if you've already updated DNS)**

If you've migrated domains but need to roll back to Vercel:

```bash
# 1. Identify which DNS records need to be changed back
# For each domain, update DNS records:

# If using CNAME record:
# Change: myapp.com -> CNAME -> my-app-abc123.ondigitalocean.app
# Back to: myapp.com -> CNAME -> cname.vercel-dns.com

# If using A records:
# Remove App Platform A records (162.159.140.98, 172.66.0.96)
# Restore original Vercel A records (check your DNS provider's history)
```

**Steps:**
1. Log into your DNS provider (Cloudflare, Namecheap, etc.)
2. Restore DNS records to point back to Vercel
3. Wait for DNS propagation (can take up to 48 hours, but often faster)
4. Monitor traffic shifting back to Vercel
5. Keep App Platform app running until you're ready to delete it

**Timeline**: DNS propagation typically takes 15 minutes to 48 hours depending on TTL settings.

---

**Rollback Option B: App Platform Deployment Rollback**

If App Platform is working but a recent deployment introduced bugs:

```bash
# 1. View deployment history
doctl apps list-deployments <app-id>

# Output will show:
# ID            Status   Created At
# abc123        ACTIVE   2025-01-15 10:30:00
# def456        ACTIVE   2025-01-14 08:15:00  <- previous working deployment
# ghi789        ACTIVE   2025-01-13 14:20:00

# 2. Rollback to previous deployment
doctl apps create-deployment <app-id> --deployment-id <previous-deployment-id>

# Example:
# doctl apps create-deployment abc-123-app --deployment-id def456
```

**When to use this:**
- Recent code push broke the app
- Need to quickly return to last known good state
- Issue is with application code, not infrastructure

**Limitations:**
- Only rolls back application code, not infrastructure changes (database schema, env vars)
- Can only rollback to deployments that are still available (typically last 10 deployments)
- Database changes are NOT rolled back automatically - handle separately

---

**Rollback Option C: Code Rollback via Git**

If you need to revert code changes:

```bash
# 1. Identify the problematic commit
git log --oneline

# 2. Revert the commit (creates new commit that undoes changes)
git revert <commit-hash>

# 3. Push to main branch
git push origin main

# App Platform will automatically deploy the reverted code
```

**When to use this:**
- Need to undo specific code changes
- Want to maintain git history
- Problem is in application code

**Alternative - Reset to previous commit** (more destructive):
```bash
# WARNING: This rewrites history
git reset --hard <previous-commit-hash>
git push origin main --force

# App Platform will deploy the previous state
```

⚠️ **Use `--force` with caution** - only do this if you understand the implications.

---

**Rollback Option D: Full App Deletion and Vercel Restoration**

If App Platform isn't working and you need to fully abort the migration:

1. **Do NOT delete Vercel app yet** - keep it running
2. Update DNS to point back to Vercel (see Option A above)
3. Wait for DNS propagation
4. Verify traffic is back on Vercel
5. Once confirmed stable, delete App Platform app:
   ```bash
   doctl apps delete <app-id>
   ```
6. Note: Databases must be deleted separately:
   ```bash
   doctl databases delete <db-id>
   ```

---

**Database Rollback Considerations:**

⚠️ **Database schemas and data are NOT automatically rolled back**

If you ran database migrations on App Platform:

**Option 1: Restore from backup**
```bash
# List database backups
doctl databases backups list <db-id>

# Restore from backup
doctl databases restore <db-id> --backup-id <backup-id>
```

**Option 2: Run reverse migrations**
- If using Prisma, Drizzle, TypeORM, etc., run down/reverse migrations
- This requires having written reverse migration scripts

**Option 3: Manual database restoration**
- If you have a pg_dump from before migration, restore it
- This is why taking a backup before migration is critical

---

**Monitoring During Rollback:**

While rolling back, monitor:
- **DNS propagation**: Use `dig myapp.com` or online DNS checkers
- **Traffic**: Check which servers are receiving traffic
- **Error rates**: Monitor both Vercel and App Platform
- **User impact**: Communicate with users if necessary

---

**Best Practices to Minimize Rollback Risk:**

1. ✅ **Test thoroughly** before updating DNS
2. ✅ **Use low TTL** on DNS records during migration (e.g., 300 seconds / 5 minutes)
3. ✅ **Backup database** before running migrations
4. ✅ **Deploy to App Platform dev environment first** (create a separate dev app)
5. ✅ **Keep Vercel running** for at least 48 hours after DNS update
6. ✅ **Monitor metrics** closely for first 24-48 hours
7. ✅ **Have team available** during DNS cutover window
8. ✅ **Communicate with users** about potential brief disruption

---

**When to Rollback:**

Consider rolling back if:
- ❌ Critical functionality is broken
- ❌ Performance is significantly degraded
- ❌ Database connection issues persist
- ❌ Authentication/authorization fails
- ❌ Third-party integrations break
- ❌ Error rate exceeds acceptable threshold

Don't rollback for:
- ✅ Minor UI issues (can be fixed with quick patch)
- ✅ Non-critical features not working (can be fixed iteratively)
- ✅ Expected differences in platform behavior (document for users)

### Step 4: Migration Summary

After completing the migration, provide:

1. **Summary table** with:
   - Original Vercel app name
   - New App Platform app name
   - Status (Success/Auto-Fixed/Failed)
   - If Auto-Fixed: what error was found and what fix was applied
   - App Platform URL
   - Domains that need DNS updates (with status: pending/completed)
   - Any manual steps required

2. **Auto-fix summary** (if automatic fixes were applied):
   - Detail the errors encountered
   - Show the fixes that were successfully applied
   - Highlight any configuration changes made

3. **Domain migration checklist:**
   - List all domains that need DNS updates
   - Provide the specific DNS changes for each domain
   - Remind user to test first, then update DNS
   - Warn about DNS propagation time (up to 48 hours)

3. **Feature gaps to address:**
   - List any Vercel features that couldn't be directly migrated
   - Provide workarounds or alternative solutions

4. **Next steps:**
   - DNS update instructions (with specific records for each domain)
   - Testing recommendations
   - Optimization suggestions
   - Timeline: Remind user to keep Vercel running until DNS fully propagates

## App Platform Strengths and When to Use It

While this guide focuses on migration considerations and limitations, it's important to understand App Platform's advantages:

**Built-in Managed Features:**
- ✅ **Managed databases** with automatic backups, point-in-time recovery, and automatic failover (PostgreSQL, Redis/ValKey)
- ✅ **Automatic SSL certificates** for all custom domains with auto-renewal
- ✅ **Built-in CDN** for global content delivery with automatic edge caching
- ✅ **Native cron jobs** and background workers for scheduled tasks
- ✅ **One-click rollbacks** to previous deployments (last 10 deployments available)
- ✅ **Automatic health checks** and restarts for failing services
- ✅ **Zero-downtime deployments** with automatic traffic shifting

**Developer Experience:**
- ✅ **Buildpack auto-detection** - minimal configuration required for standard frameworks
- ✅ **Git-based deployments** - push to deploy, no complex CI/CD setup needed
- ✅ **Preview URLs** for every deployment (`.ondigitalocean.app` subdomains)
- ✅ **Integrated logs and metrics** - no external logging service needed
- ✅ **Simple, transparent pricing** - no surprise bills, predictable costs
- ✅ **Multi-component apps** - deploy frontend, API, workers in one app
- ✅ **Internal service communication** - fast, secure service-to-service calls

**Infrastructure & Operations:**
- ✅ **Managed infrastructure** - no Kubernetes knowledge required
- ✅ **Automatic scaling** (horizontal scaling for web services)
- ✅ **Resource monitoring** built into dashboard
- ✅ **Security updates** handled automatically by buildpacks
- ✅ **Compliance** - SOC 2, GDPR, HIPAA available on higher tiers
- ✅ **Multiple regions** - deploy closer to your users (NYC, SF, AMS, SGP, BLR, etc.)

**When App Platform is a Great Fit:**
- ✅ You want **managed infrastructure** without Kubernetes complexity
- ✅ Your app uses **standard frameworks** (Next.js, Express, Django, Rails, Go, etc.)
- ✅ You value **operational simplicity** over fine-grained control
- ✅ You need **cost predictability** (flat monthly pricing per service)
- ✅ You want **database management** handled for you
- ✅ You're building **monorepo** apps with multiple services
- ✅ You prefer **convention over configuration**

**When to Consider Alternatives:**
- ⚠️ You need edge computing with sub-10ms global latency
- ⚠️ You heavily rely on Vercel Edge Functions or middleware
- ⚠️ You need ISR (Incremental Static Regeneration)
- ⚠️ You require custom infrastructure or bare metal
- ⚠️ You need highly customized caching strategies
- ⚠️ You want automatic PR preview deployments (requires GitHub Actions setup)

---

## Expected Outcomes and Success Criteria

**This guide works well for:**

✅ **Standard web applications:**
- Next.js apps with API routes and database
- React/Vue/Angular SPAs with backend APIs
- Express/Fastify Node.js applications
- Full-stack monorepo applications
- Apps with straightforward build processes
- Apps using standard framework conventions

✅ **Migration scenarios:**
- Simple to moderate complexity apps
- Apps with manageable environment variables (< 50 vars)
- Standard database usage (PostgreSQL, Redis)
- Apps without heavy Vercel-specific features
- Teams comfortable with some manual configuration

**This guide may require additional work for:**

⚠️ **Complex architectures:**
- Heavy monorepo setups with custom build orchestration
- Apps with complex multi-stage builds
- Apps with custom buildpacks or Docker requirements
- Microservices with complex inter-service dependencies
- Apps with extensive custom middleware

⚠️ **Vercel-heavy applications:**
- Apps heavily dependent on Edge Functions
- Apps using Vercel middleware extensively
- Apps relying on ISR (Incremental Static Regeneration)
- Apps with complex Vercel-specific rewrites/redirects
- Apps using Vercel's built-in image optimization heavily

⚠️ **Special requirements:**
- Real-time collaborative applications with WebSocket state management
- Apps requiring GPU acceleration
- Apps with proprietary build systems
- Apps that need extensive custom caching logic
- Apps requiring sub-10ms global edge latency

**This guide is NOT suitable for:**

❌ **Non-standard infrastructure:**
- Custom infrastructure requirements (bare metal, GPU, specific hardware)
- Apps that can't use buildpacks (custom base images, etc.)
- Apps requiring Windows servers
- Apps with licensed software that can't run in containers

❌ **Extreme edge cases:**
- Real-time gaming backends with state synchronization
- Video/audio processing requiring specialized hardware
- Blockchain nodes or cryptocurrency mining
- Applications requiring root access or kernel modules

**If your app falls into these categories**, consult DigitalOcean documentation or support for custom solutions, or consider using DigitalOcean Droplets/Kubernetes for more control.

---

**Realistic Expectations:**

**What this guide will help you achieve:**
- 🎯 Deploy your app to App Platform in 30-60 minutes (simple apps)
- 🎯 Automated error detection and fix suggestions (up to 3 attempts)
- 🎯 Working app spec YAML file you can modify
- 🎯 Clear understanding of what needs manual work
- 🎯 Rollback plan if things go wrong

**What requires human intervention:**
- 👤 Understanding your app's architecture
- 👤 Making trade-off decisions (cost, features, performance)
- 👤 Implementing code changes for unsupported features
- 👤 Testing and validating functionality
- 👤 Database data migration (separate manual process)
- 👤 Monitoring production performance

**Success metrics:**
- ✅ App builds successfully on first or second attempt
- ✅ All critical functionality works on App Platform
- ✅ Performance is acceptable (within 10-20% of Vercel)
- ✅ Database connections work reliably
- ✅ Environment variables are correctly configured
- ✅ You understand how to deploy updates

## Special Considerations

### Preview Deployments
- Vercel automatic PR previews are NOT currently available in App Platform
- **Workaround:** Create a separate dev app with dev environment
- Recommend using GitHub Actions for preview deployments

### CLI Tools vs MCP Servers

**Primary Method: CLI Tools (Recommended)**
- **doctl CLI:** Primary method for App Platform operations
  - Create apps: `doctl apps create --spec app-spec.yaml`
  - Manage deployments: `doctl apps get <app-id>`
  - Configure domains: `doctl apps create-domain <app-id> --domain <domain>`
  - View logs: `doctl apps logs <app-id> --type build|run`
  - Update apps: `doctl apps update <app-id> --spec app-spec.yaml`
- **gh CLI:** For GitHub repository operations
  - Create repo: `gh repo create <name> --private --source=. --push`
  - Fork repo: `gh repo fork <owner>/<repo>`
  - View repo: `gh repo view`
- **Vercel CLI (if migrating from Vercel):** For environment variables and app details
  - Pull env vars: `vercel env pull .env.production --environment=production`
  - Inspect app: `vercel inspect <app-name>`

**Alternative Method: MCP Servers (Optional)**
- **Vercel MCP server (Optional):** Project metadata where available
- **DigitalOcean MCP server (Optional):** App Platform operations
- **Note:** CLI tools are preferred - users don't need MCP servers

### Database Migration
- Creating new databases is straightforward
- **Data migration** is complex and requires separate tooling
- Inform user they need to handle data migration separately

### Webhooks
- Vercel deployment webhooks exist
- App Platform webhooks are available but may be limited
- Check if webhooks are needed and configure accordingly

## Error Handling

If migration fails for an app:

1. **Capture the error details** from App Platform (via DigitalOcean MCP or dashboard)
2. **Analyze common issues:**

### Build Command Errors
**Symptom:** Build fails with "command not found" or "script not found"
**Cause:** Custom build command doesn't exist or buildpack couldn't auto-detect
**Fix:**
- **Prefer buildpack auto-detection:** Leave build command empty and let buildpack detect framework
- **If auto-detection fails:**
  - Check vercel.json for custom buildCommand
  - Read package.json from migration folder at `./vercel-migration/<app-name>-migration/package.json`
  - Verify the framework is in dependencies (e.g., "next", "react", "vue")
  - For monorepos: ensure source_dir is correct
- **Reference:** [App Platform Node.js buildpack docs](https://docs.digitalocean.com/products/app-platform/reference/buildpacks/nodejs/) for auto-detection behavior

### Output Directory Errors (Static Sites)
**Symptom:** "Output directory not found" or "No files to deploy"
**Cause:** The output_dir doesn't match what the build produces
**Fix:**
- Check framework documentation to find correct output directory
- **Look in .gitignore in the migration folder** - often lists build output directories
- Run the build command locally or check framework docs
- Common output directories:
  - `build` (Create React App, Vite)
  - `dist` (Vue CLI, Vite, Angular)
  - `out` (Next.js static export)
  - `public` (Gatsby, Hugo)
  - `_site` (Jekyll)
- **Inspect the migration folder** after build to see where files are output

### Run Command Errors (Web Services)
**Symptom:** Service crashes immediately or "command not found"
**Cause:** Custom run command is invalid or buildpack couldn't auto-detect
**Fix:**
- **Prefer buildpack auto-detection:** Leave run command empty and let buildpack detect framework
- **Buildpack will automatically use:**
  - Framework-specific start commands (e.g., `next start` for Next.js)
  - The "start" script from package.json if it exists
  - The "main" field from package.json for plain Node.js apps
- **If auto-detection fails:**
  - Check vercel.json for custom start command
  - Verify app is built before starting (some frameworks need build step)
  - Ensure app listens on `process.env.PORT`
- **Reference:** [App Platform Node.js buildpack docs](https://docs.digitalocean.com/products/app-platform/reference/buildpacks/nodejs/) for default run behavior

### Port Configuration Errors
**Symptom:** Service is "unhealthy" or "failed health check"
**Cause:** App is not listening on the PORT environment variable
**Fix:**
- Ensure the app code listens on `process.env.PORT`
- Update app code if it's hardcoded to a specific port
- Default ports to check: 3000, 8080, 5000, 4000

### Monorepo/Source Directory Errors
**Symptom:** "package.json not found" or "Cannot find module"
**Cause:** The source_dir is incorrect or not set
**Fix:**
- Check Vercel's `rootDirectory` setting
- **Inspect the migration folder** at `./vercel-migration/<app-name>-migration/`
- Look for workspace configuration in the repo (lerna.json, nx.json, pnpm-workspace.yaml)
- Find where package.json exists in the repo structure
- Set source_dir to the path of the specific app (e.g., `apps/web`, `packages/api`)
- Ensure package.json exists at that path in the migration folder

### Missing Environment Variables
**Symptom:** Build succeeds but app crashes at runtime, or shows errors about missing config
**Cause:** Required environment variables weren't migrated
**Fix:**
- **Check .env.example or .env.sample in the migration folder** at `./vercel-migration/<app-name>-migration/`
- Check the app's documentation (README.md in cloned repo)
- Look for required variables in the code
- Add missing variables to App Platform
- Common missing vars: DATABASE_URL, API_KEY, NEXT_PUBLIC_*, etc.

### Dependency Installation Errors
**Symptom:** "Cannot find module" or "npm install failed"
**Cause:** Missing dependencies, lockfile issues, or version conflicts
**Fix:**
- **Check migration folder** at `./vercel-migration/<app-name>-migration/` for package-lock.json or yarn.lock
- Ensure lockfile is in the correct directory (root or source_dir)
- Check package.json for "engines" requirements
- Verify Node.js version compatibility (reference App Platform buildpack docs)
- May need to add environment variables for private npm registries
- If using workspaces (monorepo), ensure workspace configuration is correct

3. **Suggest fixes** based on error type
4. **Ask user** if they want to retry with corrections
5. **Update the app configuration** and redeploy

## User Communication

Throughout the migration:
- **Be transparent** about what features can and cannot be migrated
- **Warn early** about features requiring code changes (middleware, edge functions)
- **Provide alternatives** for unsupported features
- **Estimate timeframes** - typical migration should complete in under 30 minutes per app
- **Explain the process:**
  - Migration folders will be created at `./vercel-migration/<app-name>-migration/` for each app
  - App specs will be saved for each app as backup
  - App Platform buildpack documentation will be referenced for best practices
- **Explain assisted troubleshooting** - let users know that if deployments fail, you will analyze logs and suggest fixes for them to review
- **Show your work** - when fixes are suggested and applied, explain what error was found and what solution was proposed
- **Confirm** before making changes that could affect production
- **Keep user informed** during retry attempts (e.g., "Deployment failed, analyzing logs and suggesting fix 1 of 3...")

## Final Checklist

Before marking deployment complete:

- [ ] Local testing completed (Step 0) - dependencies install, build succeeds
- [ ] Repository is set up and accessible (Step 1)
- [ ] Database is created (if needed) and connection string is available
- [ ] Environment variables are collected and prepared
- [ ] App spec YAML file is created and validated
- [ ] The app has been created in App Platform using `doctl apps create`
- [ ] Deployment is successful (either on first try or after assisted troubleshooting)
- [ ] Assisted troubleshooting was attempted if deployment failed
- [ ] User has been informed of any fixes that were suggested and applied
- [ ] Environment variables have been configured in App Platform
- [ ] Build configurations match validated settings from repository inspection
- [ ] App spec has been saved to `app-spec.yaml` (or migration folder)
- [ ] Custom domains have been added to the App Platform app (if applicable)
- [ ] User has received DNS update instructions for each domain (if applicable)
- [ ] User understands to test before updating DNS
- [ ] User has been informed about unsupported features
- [ ] User has testing instructions
- [ ] User knows how to rollback if needed
- [ ] If migrating from Vercel: User knows to keep Vercel running until DNS propagates
- [ ] Summary includes the app's status (Success/Auto-Fixed/Failed) with details
- [ ] If successful: provide app name, URL (default ingress), and dashboard URL (`https://cloud.digitalocean.com/apps/$app_id`)
- [ ] If failed: tell the user to create a support ticket and link to DigitalOcean's documentation
- [ ] Remind user they can repeat this process for additional apps they want to deploy

---

## Key Points from This Update

1. **Multiple deployment scenarios supported:**
   - Migrating from existing Vercel deployment
   - Deploying source code directly (no Vercel required)
   - Deploying without repository access (create new repo or deploy from local)

2. **CLI tools are primary method:**
   - `doctl` for App Platform operations
   - `gh` CLI for GitHub repository operations
   - `vercel` CLI only if migrating from Vercel
   - MCP servers are optional, not required

3. **Local testing before deployment:**
   - Test build locally to catch issues early
   - Avoid waiting for deployment errors
   - Verify dependencies, build, and configuration

4. **Goal is to get app working on App Platform:**
   - Doesn't require existing Vercel deployment
   - Focus on making source code work on App Platform
   - Support users who don't want to give repo access

---

**Remember:** This is an AI-assisted deployment guide. Always prioritize user confirmation for critical decisions and be clear about what can and cannot be automated. The AI assistant provides suggestions and automates routine tasks, but the human user makes final decisions and validates results. The goal is to get the application working on App Platform through active collaboration, whether it's a migration from Vercel or a fresh deployment.
