#!/usr/bin/env bun
import { Command } from 'commander';
import { createCommand } from './commands/create.ts';
import { deleteCommand } from './commands/delete.ts';
import { execCommand } from './commands/exec.ts';
import { listCommand } from './commands/list.ts';

const program = new Command();

program
  .name('alcless')
  .description('Manage isolated macOS user sessions with Bun')
  .version('1.0.0');

program
  .command('create')
  .description('Create one or more user sessions')
  .argument('<names...>', 'names of the instances')
  .option('-t, --tools <tools>', 'additional tools to install (comma separated)')
  .option('-c, --concurrency <number>', 'number of parallel setups', '2')
  .action(createCommand);

program
  .command('list')
  .description('List all active user sessions')
  .action(listCommand);

program
  .command('delete')
  .description('Delete one or more user sessions')
  .argument('<names...>', 'names of the instances')
  .option('-c, --concurrency <number>', 'number of parallel deletions', '4')
  .action(deleteCommand);

program
  .command('exec')
  .description('Run a command or open a shell in a session')
  .argument('<name>', 'name of the instance')
  .argument('[command...]', 'command to run')
  .action(execCommand);

program.parse();
