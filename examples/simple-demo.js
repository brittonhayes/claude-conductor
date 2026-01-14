#!/usr/bin/env node

/**
 * Simple demonstration of Claude Conductor
 *
 * This script shows how to use the orchestrator to distribute tasks
 * across workers in a straightforward way.
 */

import { Orchestrator } from '../bin/orchestrator.js';

async function main() {
  console.log('Claude Conductor - Simple Demo\n');
  console.log('This demo will:');
  console.log('1. Create 3 workers');
  console.log('2. Assign 3 parallel tasks');
  console.log('3. Wait for completion');
  console.log('4. Display results\n');

  // Create orchestrator with 3 workers
  const orch = new Orchestrator(3);

  // Define some example tasks
  const tasks = [
    {
      prompt: 'List all JavaScript files in the current directory tree. Count them and show the total.',
      context: { type: 'file-listing' }
    },
    {
      prompt: 'Check if there is a package.json file. If so, list all dependencies.',
      context: { type: 'package-check' }
    },
    {
      prompt: 'Search for any TODO or FIXME comments in the code. Report how many you find.',
      context: { type: 'todo-search' }
    }
  ];

  console.log('Starting parallel execution...\n');

  // Execute tasks in parallel
  const results = await orch.executeTasks(tasks);

  // Display results
  console.log('\n' + '='.repeat(70));
  console.log('RESULTS');
  console.log('='.repeat(70) + '\n');

  for (let i = 0; i < results.length; i++) {
    const result = results[i];

    console.log(`Task ${i + 1}:`);
    console.log(`  Worker: ${result.workerId}`);
    console.log(`  Status: ${result.status}`);
    console.log(`  Type: ${result.task.context.type}`);
    console.log(`\nOutput:`);
    console.log(result.result.output);
    console.log('\n' + '-'.repeat(70) + '\n');
  }

  // Clean shutdown
  orch.shutdown();

  console.log('Demo complete!');
}

// Run the demo
main().catch(err => {
  console.error('Error:', err.message);
  process.exit(1);
});
