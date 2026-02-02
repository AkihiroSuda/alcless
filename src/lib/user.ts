import { run, sudoRun, runAsUser } from './exec.ts';
import { logger } from './logger.ts';
import { randomBytes } from 'crypto';
import { execa } from 'execa';

export interface UserInfo {
  username: string;
  instanceName: string;
}

/**
 * Generates a random secure password of specified length.
 */
function generateRandomPassword(length: number): string {
  return randomBytes(length).toString('base64').slice(0, length);
}

/**
 * Gets the current host username.
 */
export async function getHostUser(): Promise<string> {
  const { stdout } = await run('whoami', []);
  return stdout.trim();
}

/**
 * Generates the alcless username for an instance.
 */
export async function getSessionUsername(instanceName: string): Promise<string> {
  const hostUser = await getHostUser();
  // Using alcl_ to keep it short (macOS limit is 32)
  return `alcl_${hostUser}_${instanceName}`;
}

/**
 * Checks if a user exists on the system.
 */
export async function userExists(username: string): Promise<boolean> {
  try {
    await run('dscl', ['.', '-read', `/Users/${username}`], { timeoutMs: 5000 });
    return true;
  } catch {
    return false;
  }
}

/**
 * Checks if a user exists and is recognized by the system identity resolver.
 */
export async function isUserActive(username: string): Promise<boolean> {
  try {
    await run('id', ['-u', username], { timeoutMs: 5000 });
    return true;
  } catch {
    return false;
  }
}

/**
 * Checks if the network is available within the user session.
 */
async function isNetworkReady(username: string): Promise<boolean> {
  try {
    // Try to resolve github.com (common dependency)
    await runAsUser(username, 'host github.com || nslookup github.com || ping -c 1 github.com', { timeoutMs: 10000 });
    return true;
  } catch {
    return false;
  }
}

/**
 * Polls until the user is recognized by the system and network is ready.
 */
async function waitForUserReady(username: string, maxAttempts = 15): Promise<void> {
  for (let i = 0; i < maxAttempts; i++) {
    const active = await isUserActive(username);
    if (active) {
      // Once active, wait for network to stabilize
      if (await isNetworkReady(username)) return;
      logger.debug(`Waiting for network to stabilize for ${username}...`);
    } else {
      logger.debug(`Waiting for user ${username} to become active...`);
      // Flush directory service cache
      await sudoRun('dscacheutil', ['-flushcache']);
      try {
        await sudoRun('killall', ['-HUP', 'opendirectoryd']);
      } catch { /* ignore if fails */ }
    }
    await new Promise(resolve => setTimeout(resolve, 2000));
  }
  throw new Error(`User ${username} setup timed out (active: ${await isUserActive(username)}, network: ${await isNetworkReady(username)}).`);
}

/**
 * Creates a new macOS user session with a random password.
 * Uses a "Fast-Path" logic to bypass hanging sysadminctl finishing steps.
 */
export async function createSessionUser(instanceName: string): Promise<string> {
  const username = await getSessionUsername(instanceName);
  
  const existsInDscl = await userExists(username);
  const active = await isUserActive(username);

  if (existsInDscl && active) {
    logger.debug(`User ${username} already exists and is active.`);
    return username;
  }

  if (!existsInDscl) {
    logger.info(`Provisioning macOS user account: ${username}...`);
    const pw = generateRandomPassword(64);
    
    // Launch sysadminctl with inherited stdio to handle TCC prompts correctly.
    // We don't await it forever because it hangs on housekeeping.
    const subprocess = execa('sudo', ['sysadminctl', '-addUser', username, '-password', pw], {
      stdio: 'inherit',
    });

    // Race: Wait for the user record to appear in Directory Service
    let resolved = false;
    for (let i = 0; i < 40; i++) { // 20 seconds max
      if (await userExists(username)) {
        resolved = true;
        break;
      }
      await new Promise(r => setTimeout(r, 500));
    }

    if (!resolved) {
      subprocess.kill('SIGKILL');
      throw new Error(`Failed to create user record for ${username} within 20s. Please check for a system popup.`);
    }

    // Kill the hanging sysadminctl process - we have what we need (the record)
    setTimeout(() => {
      try { subprocess.kill('SIGKILL'); } catch { /* ignore */ }
    }, 1000);

    // Force system cache update
    await sudoRun('dscacheutil', ['-flushcache']);
    try { await sudoRun('killall', ['-HUP', 'opendirectoryd']); } catch { /* ignore */ }
    
    logger.success(`User identity ${username} captured.`);
  }
  
  // Wait for the system to recognize the new user and network to be ready
  await waitForUserReady(username);

  // Ensure the home directory is correctly owned and restricted
  const homeDir = `/Users/${username}`;
  logger.info(`Fixing home directory permissions for ${username}...`);
  await sudoRun('chown', [`${username}:staff`, homeDir]);
  await sudoRun('chmod', ['700', homeDir]);
  
  return username;
}

/**
 * Deletes a macOS user session.
 */
export async function deleteSessionUser(instanceName: string): Promise<void> {
  const username = await getSessionUsername(instanceName);
  
  if (!(await userExists(username))) {
    logger.debug(`User ${username} does not exist.`);
    return;
  }

  // -deleteUser <name> deletes the user and home directory
  await sudoRun('sysadminctl', ['-deleteUser', username]);
}

/**
 * Lists all alcless sessions for the current host user.
 */
export async function listSessions(): Promise<UserInfo[]> {
  const hostUser = await getHostUser();
  const prefix = `alcl_${hostUser}_`;
  
  try {
    const { stdout } = await run('dscl', ['.', '-list', '/Users']);
    const users = stdout.split('\n')
      .filter(u => u.startsWith(prefix))
      .map(username => ({
        username,
        instanceName: username.substring(prefix.length)
      }));
    
    return users;
  } catch (err: any) {
    logger.error(`Failed to list users: ${err.message}`);
    return [];
  }
}
