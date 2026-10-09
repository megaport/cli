# Portal Integration Guide

The Megaport CLI has been compiled to WebAssembly and packaged as Vue 3 components for portal integration.

**What you get:**

- Native CLI functionality in the browser
- No backend required
- TypeScript support
- Works with existing portal auth
- Full test coverage

---

## What's Included

### WebAssembly Binary

- `megaport.wasm` (~2-5 MB) - Complete CLI compiled to WASM
- `wasm_exec.js` (~15 KB) - Go WASM runtime
- Runs entirely in the browser

### Vue 3 Components

```
frontend-integration/
├── components/
│   └── MegaportTerminal.vue       # Ready-to-use terminal component
├── composables/
│   └── useMegaportWASM.ts         # Core WASM integration composable
├── types/
│   └── megaport-wasm.d.ts         # TypeScript definitions
├── utils/
│   └── type-guards.ts             # Runtime type validation
└── __tests__/                      # Comprehensive test suite
```

## Quick Start

### Step 1: Copy Files to Your Project

```bash
# In your Megaport Portal project
mkdir -p components/megaport-cli
mkdir -p composables/megaport-cli
mkdir -p types/megaport-cli
mkdir -p public/wasm

# Copy the integration files
cp frontend-integration/components/* components/megaport-cli/
cp frontend-integration/composables/* composables/megaport-cli/
cp frontend-integration/types/* types/megaport-cli/
cp frontend-integration/utils/* utils/megaport-cli/

# Build the WASM artifacts (writes web/megaport.wasm + web/wasm_exec.js)
./wasm.sh

# Copy WASM files to public directory
cp web/megaport.wasm public/wasm/
cp -f web/wasm_exec.js public/wasm/
```

### 2. Install Dependencies

```bash
npm install xterm @xterm/addon-fit @xterm/addon-web-links
```

### 3. Use the Component

```vue
<template>
  <div class="cli-section">
    <MegaportTerminal
      wasm-path="/wasm/megaport.wasm"
      wasm-exec-path="/wasm/wasm_exec.js"
    />
  </div>
</template>

<script setup lang="ts">
import MegaportTerminal from '~/components/megaport-cli/MegaportTerminal.vue';
</script>
```

## Authentication

The CLI uses in-memory credentials (no localStorage) and integrates with your existing portal auth.

### Integration Pattern

```typescript
// In your auth middleware or store
import { useMegaportWASM } from '~/composables/megaport-cli/useMegaportWASM';

// After user logs in via portal
const { setAuth } = useMegaportWASM();

onUserLogin((credentials) => {
  setAuth(
    credentials.accessKey,
    credentials.secretKey,
    credentials.environment // 'production', 'staging', or 'development'
  );
});

// On logout
onUserLogout(() => {
  const { clearAuth } = useMegaportWASM();
  clearAuth();
});
```

### Security

**Important:** Both Access Key and Secret Key are sensitive:

- Use `type="password"` for both input fields
- Never log credentials
- Call `clearAuth()` on logout
- Use HTTPS in production

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Megaport Portal (Vue 3)                │
│                                                         │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Your Portal Components                          │  │
│  │  (Dashboard, Resources, Settings, etc.)          │  │
│  └───────────────────┬──────────────────────────────┘  │
│                      │                                  │
│                      ▼                                  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  MegaportTerminal.vue                            │  │
│  │  (Reusable CLI Terminal Component)               │  │
│  └───────────────────┬──────────────────────────────┘  │
│                      │                                  │
│                      ▼                                  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  useMegaportWASM() Composable                    │  │
│  │  - WASM initialization                           │  │
│  │  - Command execution                             │  │
│  │  - Auth management                               │  │
│  │  - Error handling                                │  │
│  └───────────────────┬──────────────────────────────┘  │
│                      │                                  │
│                      ▼                                  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Browser WebAssembly Runtime                     │  │
│  │  ┌────────────────────────────────────────────┐  │  │
│  │  │  megaport.wasm (Go WASM Binary)            │  │  │
│  │  │  - Browser-compatible subset of the CLI    │  │  │
│  │  │  - Megaport API integration                │  │  │
│  │  │  - Resource modules: ports, vxc, mcr, mve, │  │  │
│  │  │    locations, partners, servicekeys        │  │  │
│  │  └────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                      │
                      ▼
          ┌──────────────────────┐
          │  Megaport API        │
          │  (REST endpoints)    │
          └──────────────────────┘
```

## Output Streaming

Command narrative (progress lines, echoes, warnings, validation errors) streams
to the host as the command runs, instead of arriving only when it completes.
Register a handler with `registerOutputHandler(callback)`; the callback is
invoked with each output chunk as it is written.

```typescript
const { registerOutputHandler } = useMegaportWASM();

registerOutputHandler((chunk) => {
  // Chunks use `\n` line endings; xterm needs `\r\n`.
  terminal.write(chunk.replace(/\n/g, '\r\n'));
});
```

**Completion contract (avoid double-rendering).** When an output handler is
registered:

- Narrative output is delivered live through the handler and is **not** repeated
  in the `execute()` result.
- The result's `output` then carries **only** structured document output
  (JSON/CSV/XML/table), or is empty when the command produced only streamed
  narrative.
- So render the streamed chunks as they arrive, and render `result.output`
  (the structured document) once at completion. Do not render both for the same
  content.

Exception: if the handler throws or delivers nothing, the WASM side disables
streaming for the rest of that command and `result.output` falls back to the
full captured narrative, so already-streamed chunks may appear there too. This
keeps output from being lost when a handler misbehaves.

When no handler is registered, `result.output` falls back to the full captured
output at completion, preserving the original non-streaming behavior.

`MegaportTerminal.vue` implements this pattern end-to-end.

## Integration Examples

### Admin CLI Page

```vue
<!-- pages/admin/cli.vue -->
<template>
  <AdminLayout>
    <PageHeader title="CLI Terminal" />
    <MegaportTerminal
      wasm-path="/wasm/megaport.wasm"
      wasm-exec-path="/wasm/wasm_exec.js"
      @command-executed="trackCliUsage"
    />
  </AdminLayout>
</template>

<script setup lang="ts">
const trackCliUsage = (command: string, result: any) => {
  analytics.track('cli_command', { command, success: !result.error });
};
</script>
```

### Dashboard Actions

```vue
<template>
  <DashboardCard title="Quick Actions">
    <button @click="createVXC">Create VXC</button>
    <button @click="listPorts">View Ports</button>
  </DashboardCard>
</template>

<script setup lang="ts">
import { useMegaportWASM } from '~/composables/megaport-cli/useMegaportWASM';

const { execute } = useMegaportWASM();

const createVXC = async () => {
  // Show your custom VXC creation wizard
  const params = await showVXCWizard();

  // Execute via CLI
  const result = await execute(`vxc buy --json '${JSON.stringify(params)}'`);

  if (result.error) {
    showError(result.error);
  } else {
    showSuccess('VXC created successfully!');
    refreshDashboard();
  }
};

const listPorts = async () => {
  const result = await execute('ports list --output json');
  const ports = JSON.parse(result.output || '[]');
  showPortsModal(ports);
};
</script>
```

### Resource Actions

```vue
<template>
  <ResourceDetail :resource="port">
    <template #actions>
      <button @click="lockPort">Lock Port</button>
      <button @click="unlockPort">Unlock Port</button>
      <button @click="checkVlan">Check VLAN</button>
    </template>
  </ResourceDetail>
</template>

<script setup lang="ts">
const props = defineProps<{ port: Port }>();
const { execute } = useMegaportWASM();

const lockPort = async () => {
  await execute(`ports lock ${props.port.uid}`);
  refresh();
};

const unlockPort = async () => {
  await execute(`ports unlock ${props.port.uid}`);
  refresh();
};

const checkVlan = async () => {
  const vlan = await prompt('Enter VLAN ID:');
  const result = await execute(`ports check-vlan ${props.port.uid} ${vlan}`);
  showResult(result);
};
</script>
```

## Telemetry

Track CLI usage:

```typescript
const { execute } = useMegaportWASM({
  onTelemetry: (event) => {
    // Send to your analytics platform
    analytics.track(event.type, {
      timestamp: event.timestamp,
      duration: event.duration,
      ...event.metadata,
    });

    // Monitor for errors
    if (event.type.endsWith('_error')) {
      errorTracking.captureEvent({
        message: event.metadata?.error,
        context: { command: event.metadata?.command },
      });
    }

    // Performance monitoring
    if (event.duration && event.duration > 5000) {
      console.warn(`Slow command: ${event.type} took ${event.duration}ms`);
    }
  },
});
```

**Events:**

- Init: `wasm_init_start`, `wasm_init_success`, `wasm_init_error`
- Commands: `command_execute_start`, `command_execute_success`, `command_execute_error`
- Auth: `auth_set`, `auth_clear`
- UI: `spinner_start`, `spinner_stop`

## Testing

The integration includes comprehensive tests you can run:

```bash
cd frontend-integration
npm test                    # Run all tests
npm run test:coverage       # Run with coverage report
npm run test:ui             # Interactive test UI
```

Tests cover component lifecycle, WASM initialization, command execution, auth, telemetry, type guards, and error handling.

## Configuration

### Nuxt 3 Configuration

Update your `nuxt.config.ts`:

```typescript
export default defineNuxtConfig({
  vite: {
    optimizeDeps: {
      exclude: ['xterm', '@xterm/addon-fit', '@xterm/addon-web-links'],
    },
    server: {
      fs: {
        allow: ['..'], // If WASM files are outside public/
      },
    },
  },

  // For production builds
  nitro: {
    compressPublicAssets: true,
    publicAssets: [
      {
        dir: 'public/wasm',
        maxAge: 60 * 60 * 24 * 365, // Cache WASM for 1 year
      },
    ],
  },
});
```

### TypeScript Configuration

Ensure your `tsconfig.json` includes:

```json
{
  "compilerOptions": {
    "types": ["vite/client"],
    "moduleResolution": "bundler"
  },
  "include": ["types/**/*", "components/**/*", "composables/**/*"]
}
```

## Error Handling

The integration includes robust error handling:

```typescript
const { execute, error, isReady } = useMegaportWASM({
  maxRetries: 3, // Retry failed init 3 times
  retryDelay: 1000, // Start with 1s delay (exponential backoff)
  initTimeout: 30000, // 30s timeout for initialization
});

// Check for initialization errors
watchEffect(() => {
  if (error.value) {
    console.error('WASM initialization failed:', error.value);
    showErrorNotification({
      title: 'CLI Unavailable',
      message: 'The CLI terminal could not be loaded. Please refresh the page.',
    });
  }
});

// Handle command errors
try {
  const result = await execute('ports list');
  if (result.error) {
    console.error('Command failed:', result.error);
    showErrorNotification({
      title: 'Command Failed',
      message: result.error,
    });
  }
} catch (err) {
  console.error('Execution error:', err);
}
```

## Performance

### 1. Lazy Loading

Load WASM only when needed:

```vue
<template>
  <div>
    <button @click="showCLI = true">Open CLI Terminal</button>

    <LazyMegaportTerminal
      v-if="showCLI"
      wasm-path="/wasm/megaport.wasm"
      wasm-exec-path="/wasm/wasm_exec.js"
    />
  </div>
</template>

<script setup>
const showCLI = ref(false);
</script>
```

### Service Worker Caching

```typescript
// sw.js or your service worker
workbox.precaching.precacheAndRoute([
  { url: '/wasm/megaport.wasm', revision: 'v1.0.0' },
  { url: '/wasm/wasm_exec.js', revision: 'v1.0.0' },
]);
```

### CDN Distribution

```vue
<MegaportTerminal
  wasm-path="https://cdn.megaport.com/cli/megaport.wasm"
  wasm-exec-path="https://cdn.megaport.com/cli/wasm_exec.js"
/>
```

## Troubleshooting

**WASM fails to load**

```typescript
// Enable debug mode
const { execute, error } = useMegaportWASM({ debug: true });

// Check browser console for detailed logs
// Verify WASM files are accessible: http://localhost:3000/wasm/megaport.wasm
```

**Commands return no output**

```typescript
// Ensure auth is set
const { setAuth, getAuthInfo } = useMegaportWASM();
setAuth(accessKey, secretKey, 'production');

// Check auth status
console.log(getAuthInfo());
```

**TypeScript errors**

```bash
npm run type-check
# Ensure type definitions are in tsconfig.json: "include": ["types/**/*"]
```
