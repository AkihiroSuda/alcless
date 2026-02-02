import { createSessionUser, deleteSessionUser, isUserActive, userExists } from '../src/lib/user.ts';
import { sudoers } from '../src/lib/sudo.ts';
import { provisionSession } from '../src/lib/provision.ts';
import { logger } from '../src/lib/logger.ts';
import { ensureSudo, runAsUser, run } from '../src/lib/exec.ts';
import chalk from 'chalk';

async function runE2E() {
  const testInstances = ['e2e-test-1', 'e2e-test-2'];
  
  console.log(chalk.bold.cyan('🧪 Starting Alcless E2E Test Suite\n'));

  // Sudo heartbeat to keep the session alive
  const heartbeat = setInterval(async () => {
    try {
      await run('sudo', ['-v']);
    } catch { /* ignore */ }
  }, 45000);

  try {
    await ensureSudo();
    
    // 1. Cleanup existing tests (including any orphaned by prefix)
    console.log(chalk.blue('🧹 Phase 1: Cleaning up existing test instances...'));
    const hostUser = process.env.USER;
    const { stdout: allUsers } = await run('dscl', ['.', '-list', '/Users']);
    const orphans = allUsers.split('\n').filter(u => u.startsWith(`alcl_${hostUser}_e2e-test-`));
    
    for (const username of orphans) {
      const inst = username.split('_').pop()!;
      logger.info(`Cleaning up orphan: ${username}`);
      await sudoers.remove(inst);
      await deleteSessionUser(inst);
    }

    // 2. Sequential Creation (to avoid multiple overlapping TCC prompts)
    console.log(chalk.blue(`🚀 Phase 2: Creating ${testInstances.length} instances...`));
    for (const inst of testInstances) {
      console.log(chalk.gray(`--- Processing: ${inst} ---`));
      
      logger.info(`Creating ${inst}...`);
      await createSessionUser(inst);
      
      logger.info(`Setting up sudoers for ${inst}...`);
      await sudoers.setup(inst);
      
      logger.info(`Provisioning ${inst} with default tools (gh, jq, uv, python, bun)...`);
      await provisionSession(inst);
      
      const username = `alcl_${hostUser}_${inst}`;
      const active = await isUserActive(username);
      if (!active) throw new Error(`${username} is not active after provisioning.`);
      logger.success(`${inst} is active and provisioned.`);
    }

    // 3. Validation & Sandbox Audit
    console.log(chalk.blue('\n🔍 Phase 3: Validating session access, tools, and sandboxing...'));
    for (const inst of testInstances) {
      const username = `alcl_${hostUser}_${inst}`;
      logger.info(`Testing access for ${username}...`);
      
      const check = await runAsUser(username, 'zsh -l -c "gh --version && jq --version && uv --version && python3 --version && bun --version"');
      if (check.exitCode === 0) {
        logger.success(`Tools verified in ${inst}:\n  ${check.stdout.trim().split('\n').slice(0, 5).join('\n  ')}`);
      } else {
        throw new Error(`Validation failed for ${inst}: ${check.stderr}`);
      }

      // --- Sandbox Audit ---
      logger.info(`Running Sandbox Audit for ${inst}...`);
      
      // 1. Filesystem Isolation (Cannot read host user's home)
      const hostHome = `/Users/${hostUser}`;
      try {
        await runAsUser(username, `ls ${hostHome}`);
        throw new Error(`SANDBOX LEAK: User ${username} could read ${hostHome}`);
      } catch (err: any) {
        if (err.message.includes('Permission denied')) {
          logger.success(`Sandbox Audit [FS]: Host home is protected from ${inst}.`);
        } else {
          throw err;
        }
      }

      // 2. Privilege Isolation (Cannot run sudo)
      try {
        await runAsUser(username, 'sudo -n true');
        throw new Error(`SANDBOX LEAK: User ${username} could execute sudo.`);
      } catch (err: any) {
        if (err.message.includes('a password is required') || err.message.includes('not in the sudoers file')) {
          logger.success(`Sandbox Audit [PRIVS]: sudo is restricted in ${inst}.`);
        } else {
          throw err;
        }
      }
    }

    // 4. Cleanup
    console.log(chalk.blue('\n🧹 Phase 4: Final Cleanup...'));
    for (const inst of testInstances) {
      await sudoers.remove(inst);
      await deleteSessionUser(inst);
      logger.info(`Deleted ${inst}`);
    }

    console.log(chalk.bold.green('\n✨ E2E Test Suite Passed Successfully!'));

  } catch (err: any) {
    console.error(chalk.red(`\n❌ E2E Test Failed: ${err.message}`));
    if (err.stack) {
      console.error(chalk.gray(err.stack));
    }
    process.exit(1);
  } finally {
    clearInterval(heartbeat);
  }
}

runE2E();
