/**
 * Remove all {false && (...)} disabled code blocks from AdminPanel.tsx
 */

const fs = require('fs');
const path = require('path');

const filePath = path.join(__dirname, '../src/components/admin/AdminPanel.tsx');
let content = fs.readFileSync(filePath, 'utf8');
const originalLength = content.length;
const originalLines = content.split('\n').length;

let removedCount = 0;
let totalLinesRemoved = 0;

// Find and remove all {false && ( blocks
while (true) {
  const marker = '{false && (';
  const startIdx = content.indexOf(marker);
  
  if (startIdx === -1) break;
  
  // Go back to find the start of the line (for clean removal)
  let lineStart = startIdx;
  while (lineStart > 0 && content[lineStart - 1] !== '\n') {
    lineStart--;
  }
  
  // Find the matching closing )}
  let depth = 0;
  let foundStart = false;
  let endIdx = startIdx;
  
  for (let i = startIdx; i < content.length; i++) {
    const char = content[i];
    
    if (char === '(' || char === '{') {
      depth++;
      foundStart = true;
    } else if (char === ')' || char === '}') {
      depth--;
      if (foundStart && depth === 0) {
        endIdx = i + 1;
        break;
      }
    }
  }
  
  if (endIdx > startIdx) {
    // Skip any trailing newlines
    while (endIdx < content.length && (content[endIdx] === '\n' || content[endIdx] === '\r')) {
      endIdx++;
    }
    
    const removed = content.substring(lineStart, endIdx);
    const linesRemoved = removed.split('\n').length;
    totalLinesRemoved += linesRemoved;
    
    console.log(`Removed disabled block: ${linesRemoved} lines`);
    
    content = content.substring(0, lineStart) + content.substring(endIdx);
    removedCount++;
  } else {
    break; // Safety: avoid infinite loop
  }
}

// Clean up multiple blank lines
content = content.replace(/\n{3,}/g, '\n\n');

fs.writeFileSync(filePath, content, 'utf8');

const newLength = content.length;
const newLines = content.split('\n').length;

console.log('\n=== Summary ===');
console.log(`Removed ${removedCount} disabled blocks (${totalLinesRemoved} lines)`);
console.log(`Original: ${(originalLength / 1024).toFixed(1)} KB, ${originalLines} lines`);
console.log(`New: ${(newLength / 1024).toFixed(1)} KB, ${newLines} lines`);
console.log(`Saved: ${((originalLength - newLength) / 1024).toFixed(1)} KB, ${originalLines - newLines} lines`);
