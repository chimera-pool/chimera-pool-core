/**
 * Modal Integration Script
 * Replaces inline modal code in App.tsx with component calls
 */

const fs = require('fs');
const path = require('path');

const filePath = path.join(__dirname, '../src/App.tsx');
let content = fs.readFileSync(filePath, 'utf8');
const originalLength = content.length;
const originalLines = content.split('\n').length;

// Replace Profile Modal inline code with component call
const profileModalStart = '{/* Profile Edit Modal */}';
const profileModalEnd = '      {/* Bug Report Modal */}';

const startIdx = content.indexOf(profileModalStart);
const endIdx = content.indexOf(profileModalEnd);

if (startIdx !== -1 && endIdx !== -1 && endIdx > startIdx) {
  const profileModalComponent = `{/* Profile Modal */}
      <ProfileModal 
        isOpen={showProfileModal} 
        onClose={() => setShowProfileModal(false)} 
        token={token || ''} 
        user={user} 
        showMessage={showMessage}
        onUserUpdate={setUser}
      />

      `;
  
  content = content.substring(0, startIdx) + profileModalComponent + content.substring(endIdx);
  console.log('✓ Replaced Profile Modal inline code with component call');
} else {
  console.log('⚠ Profile Modal markers not found');
}

// Replace Bug Report Modal inline code with component call
const bugModalStart = '{/* Bug Report Modal */}';
const bugModalEnd = '{/* My Bug Reports Modal */}';

const bugStartIdx = content.indexOf(bugModalStart);
const bugEndIdx = content.indexOf(bugModalEnd);

if (bugStartIdx !== -1 && bugEndIdx !== -1 && bugEndIdx > bugStartIdx) {
  const bugModalComponent = `{/* Bug Report Modal */}
      <BugReportModal 
        isOpen={showBugReportModal} 
        onClose={() => setShowBugReportModal(false)} 
        token={token || ''} 
        showMessage={showMessage}
      />

      `;
  
  content = content.substring(0, bugStartIdx) + bugModalComponent + content.substring(bugEndIdx);
  console.log('✓ Replaced Bug Report Modal inline code with component call');
} else {
  console.log('⚠ Bug Report Modal markers not found');
}

// Clean up multiple blank lines
content = content.replace(/\n{3,}/g, '\n\n');

fs.writeFileSync(filePath, content, 'utf8');

const newLength = content.length;
const newLines = content.split('\n').length;

console.log('\n=== Summary ===');
console.log(`Original: ${(originalLength / 1024).toFixed(1)} KB, ${originalLines} lines`);
console.log(`New: ${(newLength / 1024).toFixed(1)} KB, ${newLines} lines`);
console.log(`Saved: ${((originalLength - newLength) / 1024).toFixed(1)} KB, ${originalLines - newLines} lines`);
