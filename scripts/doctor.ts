import chalk from 'chalk';
import { run } from '../src/lib/exec.ts';

async function doctor() {
  console.log(chalk.bold.cyan('🩺 Alcless Doctor: System Health Check\n'));

  // 1. Check Platform
  if (process.platform !== 'darwin') {
    console.log(chalk.red('❌ Error: Alcless only supports macOS.'));
    process.exit(1);
  }
  console.log(chalk.green('✅ Platform: macOS detected.'));

  // 2. Check Sudo Cache
  try {
    await run('sudo', ['-n', 'true']);
    console.log(chalk.green('✅ Sudo: Already authenticated.'));
  } catch {
    console.log(chalk.yellow('⚠️  Sudo: Not authenticated. Run "sudo -v" to cache credentials.'));
  }

  // 3. Full Disk Access Guide
  console.log(chalk.bold.yellow('\n🛡️  How to enable Seamless Creation (Silence Popups):'));
  console.log(chalk.white('1. Open ') + chalk.bold('System Settings') + chalk.white(' (Cmd + Space).'));
  console.log(chalk.white('2. Go to ') + chalk.bold('Privacy & Security > Full Disk Access') + chalk.white('.'));
  console.log(chalk.white('3. Click the ') + chalk.bold('+') + chalk.white(' button and add ') + chalk.bold('Ghostty') + chalk.white('.'));
  console.log(chalk.white('4. Toggle the switch to ') + chalk.bold('ON') + chalk.white(' and restart Ghostty.\n'));

  console.log(chalk.cyan('Ready to go! Try running "make test" next.'));
}

doctor();
