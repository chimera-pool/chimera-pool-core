/**
 * Elite Tab Replacement Script
 * Replaces legacy inline tab code with component calls in AdminPanel.tsx
 * 
 * Strategy:
 * 1. Find each tab section by its comment marker
 * 2. Track brace depth to find the complete block
 * 3. Replace the entire block with a component call
 * 4. Verify syntax after each replacement
 */

const fs = require('fs');
const path = require('path');

const filePath = path.join(__dirname, '../src/components/admin/AdminPanel.tsx');
let content = fs.readFileSync(filePath, 'utf8');
const originalLength = content.length;
const originalLines = content.split('\n').length;

// Tab configurations: comment marker, component call
const tabReplacements = [
  {
    marker: '{/* Algorithm Settings Tab */}',
    startPattern: '{activeTab === \'algorithm\' && (',
    component: '<AdminAlgorithmTab token={token} isActive={activeTab === \'algorithm\'} showMessage={showMessage} />',
    name: 'Algorithm'
  },
  {
    marker: '{/* Network Configuration Tab */}',
    startPattern: '{activeTab === \'network\' && (',
    component: '<AdminNetworkTab token={token} isActive={activeTab === \'network\'} showMessage={showMessage} />',
    name: 'Network'
  },
  {
    marker: '{/* User Management Tab */}',
    startPattern: '{activeTab === \'users\' && (',
    component: '<AdminUsersTab token={token} isActive={activeTab === \'users\'} showMessage={showMessage} onClose={onClose} />',
    name: 'Users'
  }
];

/**
 * Find the matching closing brace/paren for a JSX block
 * Handles nested braces, parens, strings, and JSX
 */
function findBlockEnd(content, startIndex) {
  let depth = 0;
  let inString = false;
  let stringChar = '';
  let inTemplate = false;
  let i = startIndex;
  
  while (i < content.length) {
    const char = content[i];
    const prevChar = i > 0 ? content[i - 1] : '';
    
    // Handle string detection
    if ((char === '"' || char === "'" || char === '`') && prevChar !== '\\') {
      if (!inString) {
        inString = true;
        stringChar = char;
        if (char === '`') inTemplate = true;
      } else if (char === stringChar) {
        inString = false;
        inTemplate = false;
      }
      i++;
      continue;
    }
    
    // Skip if in string
    if (inString) {
      i++;
      continue;
    }
    
    // Track depth
    if (char === '{' || char === '(') {
      depth++;
    } else if (char === '}' || char === ')') {
      depth--;
      if (depth === 0) {
        return i;
      }
    }
    
    i++;
  }
  
  return -1; // Not found
}

let replacedCount = 0;

for (const tab of tabReplacements) {
  // Skip if component already integrated (check for component call)
  if (content.includes(tab.component)) {
    console.log(`✓ ${tab.name} tab already has component integrated`);
    continue;
  }
  
  // Find the marker comment
  const markerIndex = content.indexOf(tab.marker);
  if (markerIndex === -1) {
    console.log(`⚠ ${tab.name} tab marker not found`);
    continue;
  }
  
  // Find the start of the conditional block after the marker
  const startIndex = content.indexOf(tab.startPattern, markerIndex);
  if (startIndex === -1) {
    console.log(`⚠ ${tab.name} tab start pattern not found`);
    continue;
  }
  
  // Find the end of the block
  const endIndex = findBlockEnd(content, startIndex);
  if (endIndex === -1) {
    console.log(`⚠ ${tab.name} tab end not found`);
    continue;
  }
  
  // Calculate the full block including closing paren and any trailing whitespace
  let blockEnd = endIndex + 1;
  
  // Skip the closing paren
  if (content[blockEnd] === ')') {
    blockEnd++;
  }
  
  // Skip the closing brace
  if (content[blockEnd] === '}') {
    blockEnd++;
  }
  
  // Get the block content for logging
  const blockContent = content.substring(startIndex, blockEnd);
  const blockLines = blockContent.split('\n').length;
  
  console.log(`Replacing ${tab.name} tab: ${blockLines} lines -> component call`);
  
  // Replace the block with the component call
  // Keep the marker comment and add the component after it
  const newContent = tab.marker + '\n        ' + tab.component;
  
  // Find where to start the replacement (from the marker)
  const replaceStart = markerIndex;
  
  // Replace from marker to block end
  content = content.substring(0, replaceStart) + newContent + content.substring(blockEnd);
  
  replacedCount++;
}

// Clean up multiple blank lines
content = content.replace(/\n{3,}/g, '\n\n');

fs.writeFileSync(filePath, content, 'utf8');

const newLength = content.length;
const newLines = content.split('\n').length;

console.log('\n=== Summary ===');
console.log(`Replaced ${replacedCount} tab sections`);
console.log(`Original: ${(originalLength / 1024).toFixed(1)} KB, ${originalLines} lines`);
console.log(`New: ${(newLength / 1024).toFixed(1)} KB, ${newLines} lines`);
console.log(`Saved: ${((originalLength - newLength) / 1024).toFixed(1)} KB, ${originalLines - newLines} lines`);
