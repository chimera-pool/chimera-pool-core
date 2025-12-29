/**
 * Script to remove disabled legacy code blocks from AdminPanel.tsx
 * Removes all {false && activeTab === '...' && (...)} blocks
 */

const fs = require('fs');
const path = require('path');

const filePath = path.join(__dirname, '../src/components/admin/AdminPanel.tsx');
let content = fs.readFileSync(filePath, 'utf8');
const originalLength = content.length;
const originalLines = content.split('\n').length;

// Find all {false && activeTab === patterns and remove them
const patterns = [
  'algorithm',
  'network', 
  'roles',
  'bugs',
  'miners'
];

let removedCount = 0;

for (const tabName of patterns) {
  const marker = `{false && activeTab === '${tabName}' && (`;
  let startIdx = content.indexOf(marker);
  
  if (startIdx === -1) continue;
  
  // Find the line start (go back to find the comment and newline before)
  let lineStart = startIdx;
  // Go back to capture any whitespace before the {false
  while (lineStart > 0 && content[lineStart - 1] !== '\n') {
    lineStart--;
  }
  
  // Now find the matching closing )}
  let depth = 0;
  let inString = false;
  let stringChar = '';
  let foundStart = false;
  let endIdx = startIdx;
  
  for (let i = startIdx; i < content.length; i++) {
    const char = content[i];
    const prevChar = i > 0 ? content[i - 1] : '';
    
    // Handle string detection (simplified - doesn't handle all edge cases but good enough)
    if ((char === '"' || char === "'" || char === '`') && prevChar !== '\\') {
      if (!inString) {
        inString = true;
        stringChar = char;
      } else if (char === stringChar) {
        inString = false;
      }
      continue;
    }
    
    if (inString) continue;
    
    if (char === '{' || char === '(') {
      depth++;
      foundStart = true;
    } else if (char === '}' || char === ')') {
      depth--;
      if (foundStart && depth === 0) {
        endIdx = i + 1;
        break;
      }
    }
  }
  
  if (endIdx > startIdx) {
    // Also remove any trailing newline
    while (endIdx < content.length && (content[endIdx] === '\n' || content[endIdx] === '\r')) {
      endIdx++;
    }
    
    const removed = content.substring(lineStart, endIdx);
    console.log(`Removing ${tabName} legacy block: ${removed.length} chars, ${removed.split('\n').length} lines`);
    
    content = content.substring(0, lineStart) + content.substring(endIdx);
    removedCount++;
  }
}

// Clean up multiple blank lines
content = content.replace(/\n{3,}/g, '\n\n');

fs.writeFileSync(filePath, content, 'utf8');

const newLength = content.length;
const newLines = content.split('\n').length;

console.log('\n=== Summary ===');
console.log(`Removed ${removedCount} legacy blocks`);
console.log(`Original: ${(originalLength / 1024).toFixed(1)} KB, ${originalLines} lines`);
console.log(`New: ${(newLength / 1024).toFixed(1)} KB, ${newLines} lines`);
console.log(`Saved: ${((originalLength - newLength) / 1024).toFixed(1)} KB, ${originalLines - newLines} lines`);
