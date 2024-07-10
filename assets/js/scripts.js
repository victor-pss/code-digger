function showGif() {
  document.getElementById('loading').classList.add('show');
  document.getElementById('loading').classList.remove('hidden');
  document.getElementById('ftp-results').classList.add('hidden');
  document.getElementById('ftp-results').classList.remove('show');
  document.getElementById('no-results').classList.remove('hidden');
  document.getElementById('no-results').classList.add('show');
}

function removeGif() {
  document.getElementById('loading').classList.remove('show');
  document.getElementById('loading').classList.add('hidden');
  document.getElementById('ftp-results').classList.remove('hidden');
  document.getElementById('ftp-results').classList.add('show');
  document.getElementById('no-results').classList.remove('show');
  document.getElementById('no-results').classList.add('hidden');
}
