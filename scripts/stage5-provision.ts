import { provisionSession } from '../src/lib/provision.ts';
import { getSessionUsername } from '../src/lib/user.ts';
import { runAsUser } from '../src/lib/exec.ts';
import { logger } from '../src/lib/logger.ts';
import chalk from 'chalk';

async function stage5() {
  const TEST_NAME = 'stage-test';
  const username = await getSessionUsername(TEST_NAME);

  console.log(chalk.bold.cyan('🛠️ Stage 5: Toolchain Guarantee\n'));

  try {
    logger.info(`Provisioning tools (gh, jq, uv, python, bun) for ${username}...`);
    await provisionSession(TEST_NAME);
    logger.success('Toolchain provisioned successfully.');

    logger.info('Verifying tools via login shell...');
    // We use a login shell to ensure profile files are sourced
    const verify = await runAsUser(username, 'zsh -l -c "gh --version && jq --version && uv --version && python3 --version && bun --version"');
    
    logger.success(`Tools verified:\n${verify.stdout.trim().split('\n').filter(line => line.length > 0).slice(0, 5).map(s => '  ' + s).join('\n')}`);

    console.log(chalk.bold.green('\n✅ Stage 5 Passed: Full toolchain is provisioned and isolated.'));
  } catch (err: any) {
    logger.error(`Stage 5 Failed: ${err.message}`);
    process.exit(1);
  }
}

stage5();
