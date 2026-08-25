const trials = document.querySelector('#trials');
const status = document.querySelector('#form-status');

async function loadTrials() {
  trials.textContent = '正在读取试验…';
  try {
    const response = await fetch('/api/trials');
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    const items = (await response.json()) || [];
    if (!items.length) {
      trials.textContent = '还没有试验，请先创建一条。';
      return;
    }
    trials.replaceChildren(...items.map((trial) => {
      const article = document.createElement('article');
      article.className = 'trial';
      article.innerHTML = `<strong>${escapeHTML(trial.code)}</strong><span>${escapeHTML(trial.name)} · ${escapeHTML(trial.species)}</span><em>${escapeHTML(trial.state)}</em>`;
      return article;
    }));
  } catch (error) {
    trials.textContent = `读取失败：${error.message}`;
  }
}

document.querySelector('#trial-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const data = Object.fromEntries(new FormData(event.currentTarget));
  status.textContent = '正在保存…';
  try {
    const response = await fetch('/api/trials', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(data),
    });
    const result = await response.json();
    if (!response.ok) throw new Error(result.error || `HTTP ${response.status}`);
    event.currentTarget.reset();
    status.textContent = `已创建 ${result.code}`;
    await loadTrials();
  } catch (error) {
    status.textContent = `保存失败：${error.message}`;
  }
});

document.querySelector('#refresh').addEventListener('click', loadTrials);

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, (char) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));
}

loadTrials();
