#!/usr/bin/env node

/**
 * Helper utilities for using the orchestrator from within Claude Code
 *
 * This module provides high-level functions that Claude Code can use to
 * decompose and distribute work across workers.
 */

import { Orchestrator } from '../bin/orchestrator.js';

/**
 * Example: Parallel code review across multiple files
 */
export async function parallelCodeReview(files, numWorkers = 3) {
  const orchestrator = new Orchestrator(numWorkers);

  const tasks = files.map(file => ({
    prompt: `Review the code in ${file} for bugs, security issues, and style problems. Provide a concise summary.`,
    context: { file }
  }));

  console.log(`\nReviewing ${files.length} files in parallel...\n`);

  const results = await orchestrator.executeTasks(tasks);

  console.log('\n' + '='.repeat(60));
  console.log('CODE REVIEW RESULTS');
  console.log('='.repeat(60) + '\n');

  for (const result of results) {
    console.log(`File: ${result.task.context.file}`);
    console.log(`Worker: ${result.workerId}`);
    console.log(`Status: ${result.status}`);
    console.log(`\n${result.result.output}\n`);
    console.log('-'.repeat(60) + '\n');
  }

  orchestrator.shutdown();

  return results;
}

/**
 * Example: Parallel test execution across modules
 */
export async function parallelTests(testSuites, numWorkers = 3) {
  const orchestrator = new Orchestrator(numWorkers);

  const tasks = testSuites.map(suite => ({
    prompt: `Run tests for ${suite} and report results. Include pass/fail counts and any errors.`,
    context: { suite }
  }));

  console.log(`\nRunning ${testSuites.length} test suites in parallel...\n`);

  const results = await orchestrator.executeTasks(tasks);

  let totalPass = 0;
  let totalFail = 0;

  console.log('\n' + '='.repeat(60));
  console.log('TEST RESULTS');
  console.log('='.repeat(60) + '\n');

  for (const result of results) {
    console.log(`Suite: ${result.task.context.suite}`);
    console.log(`Worker: ${result.workerId}`);
    console.log(`\n${result.result.output}\n`);
    console.log('-'.repeat(60) + '\n');
  }

  orchestrator.shutdown();

  return results;
}

/**
 * Example: Sequential pipeline with dependency handling
 */
export async function sequentialPipeline(steps, numWorkers = 3) {
  const orchestrator = new Orchestrator(numWorkers);

  console.log(`\nExecuting ${steps.length} steps sequentially...\n`);

  const tasks = steps.map(step => ({
    prompt: step.prompt,
    context: step.context || {},
    stopOnError: step.stopOnError !== false
  }));

  const results = await orchestrator.executeSequential(tasks);

  console.log('\n' + '='.repeat(60));
  console.log('PIPELINE RESULTS');
  console.log('='.repeat(60) + '\n');

  for (let i = 0; i < results.length; i++) {
    const result = results[i];
    console.log(`Step ${i + 1}: ${steps[i].name || 'Unnamed'}`);
    console.log(`Worker: ${result.workerId}`);
    console.log(`Status: ${result.status}`);
    console.log(`\n${result.result.output}\n`);
    console.log('-'.repeat(60) + '\n');
  }

  orchestrator.shutdown();

  return results;
}

/**
 * Example: Decompose a large task into subtasks
 */
export async function decomposeAndExecute(mainGoal, numWorkers = 3) {
  const orchestrator = new Orchestrator(numWorkers);

  console.log(`\nMain goal: ${mainGoal}\n`);

  // This is a template - in real use, Claude Code would intelligently
  // decompose the goal based on code analysis
  console.log('Decomposing task...');
  console.log('(In real usage, this would be done by Claude Code analyzing the goal)\n');

  // Example decomposition
  const subtasks = [
    'Analyze requirements and identify components',
    'Implement core functionality',
    'Add error handling and edge cases',
    'Write tests',
    'Update documentation'
  ];

  const tasks = subtasks.map(task => ({
    prompt: task,
    context: { mainGoal }
  }));

  const results = await orchestrator.executeSequential(tasks);

  console.log('\n' + '='.repeat(60));
  console.log('DECOMPOSED TASK RESULTS');
  console.log('='.repeat(60) + '\n');

  for (let i = 0; i < results.length; i++) {
    const result = results[i];
    console.log(`Subtask ${i + 1}: ${subtasks[i]}`);
    console.log(`Status: ${result.status}`);
    console.log(`\n${result.result.output}\n`);
    console.log('-'.repeat(60) + '\n');
  }

  orchestrator.shutdown();

  return results;
}

// CLI for running examples
if (import.meta.url === `file://${process.argv[1]}`) {
  const command = process.argv[2];

  switch (command) {
    case 'review':
      parallelCodeReview(process.argv.slice(3)).catch(console.error);
      break;

    case 'test':
      parallelTests(process.argv.slice(3)).catch(console.error);
      break;

    case 'pipeline':
      console.log('Usage: node orchestrator-helper.js pipeline');
      console.log('This is an example only. Edit the script to define your pipeline.');
      break;

    default:
      console.log('Usage: node orchestrator-helper.js <command> [args...]');
      console.log('');
      console.log('Commands:');
      console.log('  review <file1> <file2> ...    - Parallel code review');
      console.log('  test <suite1> <suite2> ...    - Parallel test execution');
      console.log('  pipeline                       - Sequential pipeline example');
      console.log('');
      console.log('These are examples. Use the Orchestrator class directly for custom workflows.');
      break;
  }
}
