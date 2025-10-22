let startTime;
let timerInterval;

function validateForm() {
  // Get form fields
  const host = document.getElementById('host');
  const user = document.getElementById('user');
  const password = document.getElementById('password');
  const path = document.getElementById('path');
  const terms = document.getElementById('terms');
  
  const validationError = document.getElementById('validation-error');
  const validationMessage = document.getElementById('validation-message');
  
  // Hide any previous error messages
  validationError.classList.add('hidden');
  
  // Validate each field
  if (!host || !host.value.trim()) {
    showValidationError('Please enter the FTP Host address.');
    host.focus();
    return false;
  }
  
  if (!user || !user.value.trim()) {
    showValidationError('Please enter the FTP User.');
    user.focus();
    return false;
  }
  
  if (!password || !password.value.trim()) {
    showValidationError('Please enter the FTP Password.');
    password.focus();
    return false;
  }
  
  if (!path || !path.value.trim()) {
    showValidationError('Please enter the Search Path.');
    path.focus();
    return false;
  }
  
  if (!terms || !terms.value.trim()) {
    showValidationError('Please enter at least one search term.');
    terms.focus();
    return false;
  }
  
  return true;
}

function showValidationError(message) {
  const validationError = document.getElementById('validation-error');
  const validationMessage = document.getElementById('validation-message');
  
  if (validationMessage) {
    validationMessage.textContent = message;
  }
  
  if (validationError) {
    validationError.classList.remove('hidden');
    // Scroll to the error message
    validationError.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }
}

function startTimer() {
  startTime = Date.now();
  const timerDisplay = document.getElementById('timer-display');
  if (timerDisplay) {
    timerDisplay.classList.remove('hidden');
    timerInterval = setInterval(() => {
      const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
      timerDisplay.textContent = `Elapsed: ${elapsed}s`;
    }, 100);
  }
}

function stopTimer() {
  if (timerInterval) {
    clearInterval(timerInterval);
  }
  const timerDisplay = document.getElementById('timer-display');
  if (timerDisplay) {
    const finalTime = ((Date.now() - startTime) / 1000).toFixed(1);
    timerDisplay.classList.add('hidden');
    
    // Update the final elapsed time in results if element exists
    const elapsedTimeElement = document.getElementById('elapsed-time');
    if (elapsedTimeElement) {
      elapsedTimeElement.textContent = finalTime;
    }
  }
}

function showGif() {
  const loadingEl = document.getElementById('loading');
  const ftpResultsEl = document.getElementById('ftp-results');
  
  if (loadingEl) {
    loadingEl.classList.add('show');
    loadingEl.classList.remove('hidden');
  }
  
  if (ftpResultsEl) {
    ftpResultsEl.classList.add('hidden');
    ftpResultsEl.classList.remove('show');
  }
  
  startTimer();
}

function removeGif() {
  const loadingEl = document.getElementById('loading');
  const ftpResultsEl = document.getElementById('ftp-results');
  
  if (loadingEl) {
    loadingEl.classList.remove('show');
    loadingEl.classList.add('hidden');
  }
  
  if (ftpResultsEl) {
    ftpResultsEl.classList.remove('hidden');
    ftpResultsEl.classList.add('show');
  }
  
  stopTimer();
}
