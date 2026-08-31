/**
 * Megaport CLI WebAssembly Module - TypeScript Definitions
 * For Vue 3 + Vite Integration
 */

export interface MegaportCommandResult {
  /**
   * Captured stdout/stderr text from the command. When the token set via
   * `setAuthToken` was rejected by the API (401/403), this contains the
   * substring `"MEGAPORT_SESSION_EXPIRED"` — command failures are printed
   * text, not a distinct error channel, so the host must scan `output` for
   * the marker rather than relying on a separate field. On a match, prompt
   * for re-authentication and call `setAuthToken` again.
   */
  output?: string;
  error?: string;
}

export interface MegaportAuthInfo {
  accessKeySet: boolean;
  accessKeyPreview: string;
  secretKeySet: boolean;
  secretKeyPreview: string;
  accessTokenSet: boolean;
  accessTokenPreview: string;
  environment: string;
  apiURL: string;
  authMethod: 'token' | 'apikey' | 'none';
}

export interface MegaportBufferDump {
  stdout: string;
  stderr: string;
  direct: string;
}

/**
 * Prompt request from WASM for interactive commands
 */
export interface MegaportPromptRequest {
  id: string;
  message: string;
  type: string; // "text", "confirm", "resource"
  resourceType?: string; // for resource prompts
}

/**
 * Telemetry event types for tracking WASM operations
 */
export type TelemetryEventType =
  | 'wasm_init_start'
  | 'wasm_init_success'
  | 'wasm_init_error'
  | 'command_execute_start'
  | 'command_execute_success'
  | 'command_execute_error'
  | 'auth_set'
  | 'auth_token_set'
  | 'auth_clear'
  | 'spinner_start'
  | 'spinner_stop'
  | 'prompt_requested'
  | 'prompt_submitted'
  | 'prompt_cancelled';

/**
 * Telemetry event data
 */
export interface TelemetryEvent {
  type: TelemetryEventType;
  timestamp: number;
  duration?: number; // milliseconds
  metadata?: Record<string, any>;
}

/**
 * Telemetry callback function
 */
export type TelemetryCallback = (event: TelemetryEvent) => void;

/**
 * Global WASM interface exposed by the Megaport CLI
 * Available after WASM module initialization
 */
export interface MegaportWASM {
  /**
   * Deprecated stub kept for one release as a soft landing for hosts that
   * still detect or call it. It no longer executes commands and always
   * returns an immediate error result.
   * @param command - Ignored
   * @returns An error result pointing to executeMegaportCommandAsync
   * @deprecated Use executeMegaportCommandAsync instead; this function does not run commands
   */
  executeMegaportCommand(command: string): MegaportCommandResult;

  /**
   * Execute a CLI command asynchronously (RECOMMENDED)
   * @param command - Full command string (e.g., "port list --output json")
   * @param callback - Callback function to receive the result
   */
  executeMegaportCommandAsync(
    command: string,
    callback: (result: MegaportCommandResult) => void
  ): void;

  /**
   * Read a config file from localStorage
   * @param filename - Name of the file to read
   */
  readConfigFile(filename: string): { content?: string; error?: string };

  /**
   * Write a config file to localStorage
   * @param filename - Name of the file to write
   * @param content - Content to write
   */
  writeConfigFile(filename: string, content: string): { success: boolean };

  /**
   * Get authentication information
   */
  debugAuthInfo(): MegaportAuthInfo;

  /**
   * Save data to localStorage
   */
  saveToLocalStorage(key: string, value: string): boolean;

  /**
   * Load data from localStorage
   */
  loadFromLocalStorage(key: string): string;

  /**
   * Set authentication credentials securely (in-memory only, recommended)
   * Stores credentials in Go environment variables and JavaScript global
   * Does NOT use localStorage to prevent XSS attacks
   * @param accessKey - Megaport API access key
   * @param secretKey - Megaport API secret key
   * @param environment - Environment (production, staging, development)
   * @param managedAccountUID - Optional managed account company UID to act on
   *   behalf of; sent as the X-Call-Context header on authenticated requests
   * @returns Result object with success status
   */
  setAuthCredentials(
    accessKey: string,
    secretKey: string,
    environment: string,
    managedAccountUID?: string
  ): { success: boolean; error?: string };

  /**
   * Set authentication using an existing token from the portal session,
   * bypassing the OAuth flow. Use this when the host page already holds a
   * valid Megaport access token (typically via SSO into the portal).
   *
   * ## Environment resolution
   *
   * The environment is resolved in this order:
   *
   * 1. The explicit `environment` argument, if non-empty.
   * 2. The environment derived from `hostname` per the Megaport conventions:
   *    - `megaport.com`, `www.megaport.com`, and any `<app>.megaport.com`
   *      (single-word app, no hyphens) → `"production"`.
   *    - `<app>-<env>.megaport.com` → `<env>` (env may contain further
   *      hyphens, so `api-mpone-dev.megaport.com` resolves to `"mpone-dev"`).
   *
   * If neither yields a value (e.g. `hostname` is `"localhost"`, a private IP,
   * or a non-Megaport domain), **the call fails**. The function never
   * silently falls back to production.
   *
   * ## API URL
   *
   * The API URL is always built from the resolved environment:
   * - `"production"` → `https://api.megaport.com/`.
   * - anything else → `https://api-<env>.megaport.com/`.
   *
   * ## Validation
   *
   * The explicit `environment` argument must match `/^[a-z0-9-]+$/` — any
   * other value (containing `/`, `.`, `@`, `:`, uppercase, etc.) is rejected
   * with an error to prevent hostname injection into the API URL.
   *
   * ## Expiry and session handling
   *
   * Pass the token's real expiry as `expiry` when the host knows it (epoch
   * milliseconds, or an RFC3339 string). This is optional and backward
   * compatible: omitting it (or passing `0`/an unparseable value) is fine.
   * The CLI stores `expiry` and echoes it back in the result, but does not
   * check it proactively — there is no background timer or pre-request
   * check that watches it. The CLI also does not refresh tokens on this
   * path, since there are no credentials to refresh with. The only actual
   * signal is reactive: whenever the API rejects a request with 401/403
   * (whether because the token expired or for any other reason), that
   * command's output contains the marker `"MEGAPORT_SESSION_EXPIRED"` (see
   * {@link MegaportCommandResult.output}). The host should treat that as a
   * signal to prompt for re-authentication and call `setAuthToken` again.
   *
   * @param token - The access token from the portal session
   * @param hostname - The current hostname, e.g. `window.location.hostname`
   * @param environment - Optional explicit environment override; supersedes the hostname-derived value. Useful when `hostname` is `"localhost"` or a non-portal host, or when the portal needs to talk to a specific backend regardless of where it's served
   * @param expiry - Optional real expiry of `token`, as epoch milliseconds (number) or an RFC3339 string. Stored and echoed back, but not checked proactively (see above). Omit, or pass `0`/an unparseable value, if the expiry is unknown
   * @returns On success: `{ success: true, environment, hostname, apiURL, expiry? }` where `environment` is the resolved env name (e.g. `"qa"`), `apiURL` is the matching `api-<env>.megaport.com/` URL, and `expiry` (RFC3339 string) echoes back the resolved expiry if one was set. On failure: `{ success: false, error }` with a human-readable message; the caller should surface the message to guide the user
   *
   * @example
   * // Portal served from a recognised host — no override needed.
   * setAuthToken(token, window.location.hostname);
   *
   * @example
   * // Local development against the qa backend.
   * setAuthToken(token, window.location.hostname, "qa");
   *
   * @example
   * // Record the token's known expiry for display; the CLI still only
   * // reacts to an actual 401/403 from the API, not to this timestamp.
   * setAuthToken(token, window.location.hostname, undefined, Date.now() + 60 * 60 * 1000);
   */
  setAuthToken(
    token: string,
    hostname: string,
    environment?: string,
    expiry?: number | string
  ): { success: boolean; error?: string; environment?: string; hostname?: string; apiURL?: string; expiry?: string };

  /**
   * Clear authentication credentials from memory
   * @returns Result object with success status
   */
  clearAuthCredentials(): { success: boolean };

  /**
   * Reset WASM output buffers
   */
  resetWasmOutput(): boolean;

  /**
   * Get current WASM output
   */
  getWasmOutput(): string;

  /**
   * Toggle WASM debug mode
   */
  toggleWasmDebug(): boolean;

  /**
   * Dump all buffer contents for debugging
   */
  dumpBuffers(): MegaportBufferDump;

  /**
   * Check if WASM debug mode is enabled
   */
  wasmDebug(): boolean;

  /**
   * Log location command debug information
   */
  logLocationCommand(message: string): void;

  /**
   * Register a prompt handler for interactive commands
   * @param callback - Function to call when user input is needed
   */
  registerPromptHandler(
    callback: (request: MegaportPromptRequest) => void
  ): boolean;

  /**
   * Register a handler for live command output.
   *
   * The callback is invoked with each chunk of narrative output as the command
   * writes it. When a handler is registered the narrative is streamed here and
   * is not repeated in the command result (see MegaportCommandResult.output).
   * Chunks use `\n` line endings.
   *
   * @param callback - Function called with each output chunk
   */
  registerOutputHandler(callback: (chunk: string) => void): boolean;

  /**
   * Submit a response to a pending prompt
   * @param id - Prompt ID
   * @param response - User's response
   */
  submitPromptResponse(id: string, response: string): void;

  /**
   * Cancel a pending prompt
   * @param id - Prompt ID
   */
  cancelPrompt(id: string): void;

  /**
   * Get list of pending prompts (for debugging)
   */
  getPendingPrompts(): MegaportPromptRequest[];

  /**
   * Tell the WASM table renderer the host terminal's viewport width, in
   * columns, so table output scales to it instead of a fixed layout.
   * Call on terminal init and again on every resize (after the fit addon
   * recalculates `terminal.cols`).
   * @param cols - Terminal width in columns
   */
  setTerminalWidth(cols: number): { success: boolean; error?: string };
}

/**
 * Go WASM runtime
 */
export interface GoWASM {
  run(instance: WebAssembly.Instance): void;
  importObject: WebAssembly.Imports;
  _exitPromise?: Promise<void>;
  _resolveExitPromise?: () => void;
  _pendingEvent?: { id: number; this: any; args: any[] };
}

declare global {
  interface Window {
    executeMegaportCommand?: (command: string) => MegaportCommandResult;
    executeMegaportCommandAsync?: (
      command: string,
      callback: (result: MegaportCommandResult) => void
    ) => void;
    readConfigFile?: (filename: string) => { content?: string; error?: string };
    writeConfigFile?: (
      filename: string,
      content: string
    ) => { success: boolean };
    debugAuthInfo?: () => MegaportAuthInfo;
    saveToLocalStorage?: (key: string, value: string) => boolean;
    loadFromLocalStorage?: (key: string) => string;
    setAuthCredentials?: (
      accessKey: string,
      secretKey: string,
      environment: string,
      managedAccountUID?: string
    ) => { success: boolean; error?: string };
    setAuthToken?: (
      token: string,
      hostname: string,
      environment?: string,
      expiry?: number | string
    ) => { success: boolean; error?: string; environment?: string; hostname?: string; apiURL?: string; expiry?: string };
    clearAuthCredentials?: () => { success: boolean };
    resetWasmOutput?: () => boolean;
    getWasmOutput?: () => string;
    toggleWasmDebug?: () => boolean;
    dumpBuffers?: () => MegaportBufferDump;
    wasmDebug?: () => boolean;
    logLocationCommand?: (message: string) => void;
    registerPromptHandler?: (
      callback: (request: MegaportPromptRequest) => void
    ) => boolean;
    registerOutputHandler?: (callback: (chunk: string) => void) => boolean;
    submitPromptResponse?: (id: string, response: string) => void;
    cancelPrompt?: (id: string) => void;
    getPendingPrompts?: () => MegaportPromptRequest[];
    setTerminalWidth?: (cols: number) => { success: boolean; error?: string };
    Go?: new () => GoWASM;
    // Content-hashed wasm URL injected into index.html at build time (ESD-1272).
    __MEGAPORT_WASM_URL__?: string;
  }
}

export {};
