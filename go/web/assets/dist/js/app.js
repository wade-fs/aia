/**
 * AIA Personal Agent — Native Web Console
 * No frameworks, vanilla JS, embedded static assets.
 */

const app = {
  currentTab: 'overview',
  eventSource: null,
  activeTaskFilter: '',

  init() {
    this.bindEvents();
    this.checkSession();
  },

  async apiCall(url, options = {}) {
    options.headers = options.headers || {};
    if (options.body && typeof options.body === 'object' && !(options.body instanceof FormData)) {
      options.headers['Content-Type'] = 'application/json';
      options.body = JSON.stringify(options.body);
    }
    // Prevent CSRF by adding custom header for non-GET requests
    if (options.method && options.method !== 'GET') {
      options.headers['X-Requested-With'] = 'XMLHttpRequest';
    }

    try {
      const res = await fetch(url, options);
      if (res.status === 401) {
        this.showLogin(true);
        throw new Error('Authentication required');
      }
      const data = await res.json();
      if (!res.ok) {
        throw new Error((data.error && data.error.message) || `HTTP error ${res.status}`);
      }
      return data.data;
    } catch (err) {
      console.error(`API Error [${url}]:`, err);
      throw err;
    }
  },

  bindEvents() {
    // Nav Tab Switching
    document.querySelectorAll('#main-nav .nav-tab').forEach(tab => {
      tab.addEventListener('click', (e) => {
        const target = e.currentTarget.getAttribute('data-tab');
        this.switchTab(target);
      });
    });

    // Login button
    document.getElementById('btn-login').addEventListener('click', () => this.handleLogin());
    document.getElementById('token-input').addEventListener('keydown', (e) => {
      if (e.key === 'Enter') this.handleLogin();
    });

    // Logout
    document.getElementById('btn-logout').addEventListener('click', () => this.handleLogout());

    // Task Filter Chips
    document.querySelectorAll('[data-task-filter]').forEach(chip => {
      chip.addEventListener('click', (e) => {
        document.querySelectorAll('[data-task-filter]').forEach(c => c.classList.remove('active'));
        e.currentTarget.classList.add('active');
        this.activeTaskFilter = e.currentTarget.getAttribute('data-task-filter');
        this.loadTasks();
      });
    });

    // Create Task Modal & Form
    document.getElementById('btn-create-task-modal').addEventListener('click', () => {
      document.getElementById('create-task-modal').classList.remove('hidden');
    });

    document.getElementById('form-create-task').addEventListener('submit', async (e) => {
      e.preventDefault();
      await this.handleCreateTask();
    });

    // Investment Evaluate Form
    document.getElementById('form-evaluate-investment').addEventListener('submit', async (e) => {
      e.preventDefault();
      await this.handleEvaluateInvestment();
    });

    // Investment Sync & Calibration Buttons
    const btnSync = document.getElementById('btn-sync-market-data');
    if (btnSync) btnSync.addEventListener('click', () => this.handleSyncMarketData());

    const btnEval = document.getElementById('btn-eval-predictions');
    if (btnEval) btnEval.addEventListener('click', () => this.handleEvaluatePredictions());

    // Investment Symbol Selectors Linkage
    const selEval = document.getElementById('inv-eval-symbol');
    if (selEval) {
      selEval.addEventListener('change', (e) => this.onSymbolChange(e.target.value));
    }
    const selSync = document.getElementById('inv-sync-symbol');
    if (selSync) {
      selSync.addEventListener('change', (e) => this.onSymbolChange(e.target.value));
    }

    // Audit Query
    document.getElementById('btn-refresh-audit').addEventListener('click', () => this.loadAudit());
  },

  async checkSession() {
    try {
      const data = await this.apiCall('/api/v1/auth/session');
      document.getElementById('user-display').textContent = data.actor_id || 'Admin';
      this.showLogin(false);
      this.initSSE();
      this.switchTab(this.currentTab);
    } catch (err) {
      this.showLogin(true);
    }
  },

  showLogin(show) {
    const el = document.getElementById('login-overlay');
    if (show) {
      el.classList.remove('hidden');
      document.getElementById('token-input').value = '';
      document.getElementById('token-input').focus();
    } else {
      el.classList.add('hidden');
    }
  },

  async handleLogin() {
    const token = document.getElementById('token-input').value.trim();
    const errEl = document.getElementById('login-error');
    errEl.classList.add('hidden');

    if (!token) {
      errEl.textContent = '請輸入登入 Token';
      errEl.classList.remove('hidden');
      return;
    }

    try {
      await this.apiCall('/api/v1/auth/login', {
        method: 'POST',
        body: { token }
      });
      this.showToast('登入成功！已建立安全 Session', 'success');
      this.showLogin(false);
      this.initSSE();
      this.switchTab(this.currentTab);
    } catch (err) {
      errEl.textContent = err.message || 'Token 驗證失敗';
      errEl.classList.remove('hidden');
    }
  },

  async handleLogout() {
    try {
      await this.apiCall('/api/v1/auth/logout', { method: 'POST' });
    } catch (e) {}
    if (this.eventSource) {
      this.eventSource.close();
    }
    this.showLogin(true);
    this.showToast('已登出');
  },

  initSSE() {
    if (this.eventSource) {
      this.eventSource.close();
    }

    const indicator = document.getElementById('sse-status');
    const textEl = indicator.querySelector('.status-text');

    indicator.className = 'sse-indicator';
    textEl.textContent = '連線中...';

    const es = new EventSource('/api/v1/events');
    this.eventSource = es;

    es.onopen = () => {
      indicator.className = 'sse-indicator connected';
      textEl.textContent = '即時同步中';
    };

    es.onerror = () => {
      indicator.className = 'sse-indicator error';
      textEl.textContent = '連線中斷';
    };

    es.addEventListener('task.created', (e) => {
      this.showToast('⚡ 新任務已建立', 'info');
      this.refreshCurrent();
    });

    es.addEventListener('task.updated', (e) => {
      this.refreshCurrent();
    });

    es.addEventListener('task.waiting_approval', (e) => {
      this.showToast('⚠️ 有任務等待核准！', 'warning');
      this.refreshCurrent();
    });

    es.addEventListener('investment.updated', (e) => {
      this.showToast('📈 投資模型已更新', 'info');
      if (this.currentTab === 'investment' || this.currentTab === 'pending') {
        this.refreshCurrent();
      }
    });
  },

  switchTab(tabName) {
    this.currentTab = tabName;
    document.querySelectorAll('#main-nav .nav-tab').forEach(t => {
      t.classList.toggle('active', t.getAttribute('data-tab') === tabName);
    });
    document.querySelectorAll('.tab-pane').forEach(p => {
      p.classList.toggle('active', p.id === `tab-${tabName}`);
    });

    this.refreshCurrent();
  },

  refreshCurrent() {
    switch (this.currentTab) {
      case 'overview': this.loadOverview(); break;
      case 'tasks': this.loadTasks(); break;
      case 'pending': this.loadPending(); break;
      case 'investment': this.loadInvestment(); break;
      case 'coding': this.loadCoding(); break;
      case 'writing': this.loadWriting(); break;
      case 'memory': this.loadMemory(); break;
      case 'agents': this.loadAgents(); break;
      case 'audit': this.loadAudit(); break;
    }
    this.updatePendingCountBadge();
  },

  async updatePendingCountBadge() {
    try {
      const data = await this.apiCall('/api/v1/pending');
      const badge = document.getElementById('pending-badge');
      if (data && data.total_count > 0) {
        badge.textContent = data.total_count;
        badge.classList.remove('hidden');
      } else {
        badge.classList.add('hidden');
      }
    } catch (e) {}
  },

  // ----------------------------------------------------------------
  // 1. Overview Tab
  // ----------------------------------------------------------------
  async loadOverview() {
    try {
      const sys = await this.apiCall('/api/v1/system/status');
      document.getElementById('sys-status').textContent = sys.database === 'OK' ? 'RUNNING (Healthy)' : sys.database;
      document.getElementById('sys-schema-ver').textContent = `SQLite Schema v${sys.schema_version}`;
      document.getElementById('sys-provider').textContent = sys.provider.name;
      document.getElementById('sys-provider-status').textContent = sys.provider.status;

      // Activity Monitor
      try {
        const act = await this.apiCall('/api/v1/system/activity');
        if (act) {
          const runBadge = document.getElementById('activity-runtime-badge');
          if (runBadge) {
            runBadge.textContent = act.runtime;
            runBadge.className = act.runtime === 'RUNNING' ? 'badge badge-success' : 'badge badge-neutral';
          }
          const actRuntime = document.getElementById('act-runtime');
          if (actRuntime) {
            actRuntime.textContent = act.runtime;
            actRuntime.className = act.runtime === 'RUNNING' ? 'badge badge-success' : 'badge badge-neutral';
          }
          const schedWorker = document.getElementById('act-schedule-worker');
          if (schedWorker) {
            schedWorker.textContent = act.schedule_worker;
            schedWorker.className = act.schedule_worker === 'RUNNING' ? 'badge badge-primary' : 'badge badge-neutral';
          }
          const taskWorker = document.getElementById('act-task-worker');
          if (taskWorker) {
            taskWorker.textContent = act.task_worker;
            taskWorker.className = (act.task_worker === 'BUSY' || act.task_worker === 'RUNNING') ? 'badge badge-warning' : 'badge badge-subtle';
          }
          const lastSync = document.getElementById('act-last-data-sync');
          if (lastSync) lastSync.textContent = act.last_data_sync;
          const lastRes = document.getElementById('act-last-research');
          if (lastRes) lastRes.textContent = act.last_investment_research;
          const pendingEl = document.getElementById('act-pending-tasks');
          if (pendingEl) pendingEl.textContent = act.pending_tasks;
          const nextRun = document.getElementById('act-next-run');
          if (nextRun) nextRun.textContent = act.next_scheduled_run;
        }
      } catch (e) {
        console.warn('Activity monitor load failed', e);
      }

      // Pending count
      const pending = await this.apiCall('/api/v1/pending');
      document.getElementById('stat-pending-count').textContent = pending ? pending.total_count : 0;

      // Recent tasks
      const tasks = await this.apiCall('/api/v1/tasks?limit=5');
      const tbody = document.querySelector('#overview-tasks-table tbody');
      if (tasks && tasks.length > 0) {
        let activeCount = 0;
        tbody.innerHTML = tasks.map(t => {
          if (t.status === 'READY' || t.status === 'RUNNING') activeCount++;
          return `
            <tr>
              <td><strong>${this.escape(t.title)}</strong></td>
              <td>${this.escape(t.agent_id || 'unassigned')}</td>
              <td><span class="badge badge-${t.status}">${t.status}</span></td>
              <td>${this.formatTime(t.created_at)}</td>
            </tr>
          `;
        }).join('');
        document.getElementById('stat-active-tasks').textContent = activeCount;
      } else {
        tbody.innerHTML = '<tr><td colspan="4" class="text-center text-muted">目前沒有任務</td></tr>';
      }

      // Pending List Summary
      const pendList = document.getElementById('overview-pending-list');
      if (pending && pending.total_count > 0) {
        let html = '';
        if (pending.tasks && pending.tasks.length > 0) {
          html += pending.tasks.map(t => `
            <div class="card p-4 mt-2">
              <span class="badge badge-WAITING_APPROVAL">TASK</span>
              <strong>${this.escape(t.title)}</strong>
              <div class="mt-2">
                <button class="btn btn-sm btn-accent" onclick="app.actionPending('tasks', '${t.id}', 'approve')">核准</button>
                <button class="btn btn-sm btn-outline ml-2" onclick="app.actionPending('tasks', '${t.id}', 'reject')">拒絕</button>
              </div>
            </div>
          `).join('');
        }
        if (pending.trade_proposals && pending.trade_proposals.length > 0) {
          html += pending.trade_proposals.map(p => `
            <div class="card p-4 mt-2">
              <span class="badge badge-${p.action}">TRADE ${p.action}</span>
              <strong>${p.quantity} 股 @ $${p.estimated_price}</strong>
              <div class="mt-2">
                <button class="btn btn-sm btn-accent" onclick="app.actionPending('proposals', '${p.id}', 'approve')">核准交易</button>
                <button class="btn btn-sm btn-outline ml-2" onclick="app.actionPending('proposals', '${p.id}', 'reject')">駁回</button>
              </div>
            </div>
          `).join('');
        }
        pendList.innerHTML = html;
      } else {
        pendList.innerHTML = '<div class="text-center text-muted p-4">目前沒有等待核准的項目</div>';
      }
    } catch (e) {
      console.error(e);
    }
  },

  // ----------------------------------------------------------------
  // 2. Tasks Tab
  // ----------------------------------------------------------------
  async loadTasks() {
    try {
      let url = '/api/v1/tasks?limit=50';
      if (this.activeTaskFilter) {
        url += `&status=${this.activeTaskFilter}`;
      }
      const tasks = await this.apiCall(url);
      const tbody = document.querySelector('#tasks-table tbody');
      if (tasks && tasks.length > 0) {
        tbody.innerHTML = tasks.map(t => `
          <tr>
            <td><code class="text-muted">${t.id.substring(0, 8)}</code></td>
            <td><strong>${this.escape(t.title)}</strong><br><small class="text-muted">${this.escape(t.description || '')}</small></td>
            <td>${this.escape(t.agent_id || '-')}</td>
            <td>${t.priority}</td>
            <td><span class="badge badge-subtle">${t.risk}</span></td>
            <td><span class="badge badge-${t.status}">${t.status}</span></td>
            <td>${this.formatTime(t.updated_at || t.created_at)}</td>
            <td>
              ${t.status === 'RUNNING' ? `<button class="btn btn-sm btn-outline" onclick="app.cancelTask('${t.id}')">取消</button>` : ''}
              ${t.status === 'BLOCKED' ? `<button class="btn btn-sm btn-primary" onclick="app.resumeTask('${t.id}')">重試</button>` : ''}
              ${t.status === 'WAITING_APPROVAL' ? `
                <button class="btn btn-sm btn-accent" onclick="app.actionPending('tasks', '${t.id}', 'approve')">核准</button>
                <button class="btn btn-sm btn-outline ml-2" onclick="app.actionPending('tasks', '${t.id}', 'reject')">拒絕</button>
              ` : ''}
            </td>
          </tr>
        `).join('');
      } else {
        tbody.innerHTML = '<tr><td colspan="8" class="text-center text-muted">無符合條件的任務</td></tr>';
      }
    } catch (e) {
      console.error(e);
    }
  },

  async handleCreateTask() {
    const title = document.getElementById('new-task-title').value.trim();
    const agent = document.getElementById('new-task-agent').value;
    const desc = document.getElementById('new-task-desc').value.trim();
    const prio = parseInt(document.getElementById('new-task-priority').value, 10) || 50;
    const risk = document.getElementById('new-task-risk').value;

    try {
      await this.apiCall('/api/v1/tasks', {
        method: 'POST',
        body: {
          title,
          description: desc,
          agent_id: agent,
          priority: prio,
          risk
        }
      });
      this.showToast(`任務「${title}」已建立，狀態為 READY`, 'success');
      this.closeTaskModal();
      this.loadTasks();
    } catch (err) {
      alert('建立任務失敗: ' + err.message);
    }
  },

  closeTaskModal() {
    document.getElementById('create-task-modal').classList.add('hidden');
    document.getElementById('form-create-task').reset();
  },

  async cancelTask(id) {
    if (!confirm('確定取消此任務？')) return;
    try {
      await this.apiCall(`/api/v1/tasks/${id}/cancel`, { method: 'POST' });
      this.showToast('任務已取消');
      this.loadTasks();
    } catch (err) {
      alert(err.message);
    }
  },

  async resumeTask(id) {
    try {
      await this.apiCall(`/api/v1/tasks/${id}/resume`, { method: 'POST' });
      this.showToast('任務已恢復 (READY)');
      this.loadTasks();
    } catch (err) {
      alert(err.message);
    }
  },

  // ----------------------------------------------------------------
  // 3. Pending Review Tab
  // ----------------------------------------------------------------
  async loadPending() {
    try {
      const data = await this.apiCall('/api/v1/pending');

      // 1. Pending Tasks
      const tasksTbody = document.querySelector('#pending-tasks-table tbody');
      if (data.tasks && data.tasks.length > 0) {
        tasksTbody.innerHTML = data.tasks.map(t => `
          <tr>
            <td><strong>${this.escape(t.title)}</strong></td>
            <td>${this.escape(t.agent_id)}</td>
            <td><span class="badge badge-WAITING_APPROVAL">${t.risk}</span></td>
            <td>${this.escape(t.description || '-')}</td>
            <td>
              <button class="btn btn-sm btn-accent" onclick="app.actionPending('tasks', '${t.id}', 'approve')">核准執行</button>
              <button class="btn btn-sm btn-outline ml-2" onclick="app.actionPending('tasks', '${t.id}', 'reject')">拒絕</button>
            </td>
          </tr>
        `).join('');
      } else {
        tasksTbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">無待核准任務</td></tr>';
      }

      // 2. Pending Proposals
      const propsTbody = document.querySelector('#pending-proposals-table tbody');
      if (data.trade_proposals && data.trade_proposals.length > 0) {
        propsTbody.innerHTML = data.trade_proposals.map(p => {
          let details = {};
          try { details = JSON.parse(p.proposal_json); } catch (e) {}
          return `
            <tr>
              <td><span class="badge badge-${p.action}">${p.action}</span> <strong>${details.symbol || '-'}</strong></td>
              <td><strong>${p.quantity.toLocaleString()}</strong> 股</td>
              <td>$${p.estimated_price.toLocaleString()}</td>
              <td><strong class="text-accent">$${p.estimated_total_cost.toLocaleString()}</strong></td>
              <td><small>手續費: $${details.broker_fee || 0} | 證交稅: $${details.transaction_tax || 0}</small></td>
              <td>${this.formatTime(p.expires_at)}</td>
              <td>
                <button class="btn btn-sm btn-accent" onclick="app.actionPending('proposals', '${p.id}', 'approve')">核准交易提案</button>
                <button class="btn btn-sm btn-outline ml-2" onclick="app.actionPending('proposals', '${p.id}', 'reject')">駁回</button>
              </td>
            </tr>
          `;
        }).join('');
      } else {
        propsTbody.innerHTML = '<tr><td colspan="7" class="text-center text-muted">無待審核投資提案</td></tr>';
      }

      // 3. Pending Memory
      const memTbody = document.querySelector('#pending-memory-table tbody');
      if (data.memories && data.memories.length > 0) {
        memTbody.innerHTML = data.memories.map(m => `
          <tr>
            <td><strong>${this.escape(m.key)}</strong></td>
            <td><span class="badge badge-subtle">${m.type}</span></td>
            <td>${m.confidence.toFixed(2)}</td>
            <td>${this.escape(m.content)}</td>
            <td>
              <button class="btn btn-sm btn-accent" onclick="app.actionPending('memory', '${m.id}', 'approve')">轉為正式記憶</button>
              <button class="btn btn-sm btn-outline ml-2" onclick="app.actionPending('memory', '${m.id}', 'reject')">捨棄</button>
            </td>
          </tr>
        `).join('');
      } else {
        memTbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">無待審核記憶候選</td></tr>';
      }
    } catch (e) {
      console.error(e);
    }
  },

  async actionPending(type, id, action) {
    try {
      await this.apiCall(`/api/v1/pending/${type}/${id}/${action}`, { method: 'POST' });
      this.showToast(`項目已${action === 'approve' ? '核准' : '駁回'}`);
      this.loadPending();
      this.updatePendingCountBadge();
    } catch (err) {
      alert(err.message);
    }
  },

  // ----------------------------------------------------------------
  // 4. Investment Agent Tab
  // ----------------------------------------------------------------
  async loadInvestment() {
    try {
      const profData = await this.apiCall('/api/v1/agents/investment/profile');
      if (profData && profData.profile) {
        const p = profData.profile;
        this.currentCapitalLimit = p.capital_limit;
        document.getElementById('inv-prof-name').textContent = p.name;
        document.getElementById('inv-prof-market').textContent = `Market: ${p.market} (${p.currency})`;
        document.getElementById('inv-capital-limit').textContent = '$' + p.capital_limit.toLocaleString();
        document.getElementById('inv-capital-basis').textContent = `Basis: ${p.capital_limit_basis}`;
      }

      const snapData = await this.apiCall('/api/v1/agents/investment/snapshot');
      if (snapData && snapData.snapshot) {
        const s = snapData.snapshot;
        this.currentAvailableCash = s.available_cash;
        document.getElementById('inv-cash').textContent = '$' + s.available_cash.toLocaleString();
        document.getElementById('inv-total-value').textContent = '$' + s.total_portfolio_value.toLocaleString();

        const cashRatio = s.total_portfolio_value > 0 ? (s.available_cash / s.total_portfolio_value * 100).toFixed(1) : '100';
        document.getElementById('inv-cash-ratio').textContent = `現金比重: ${cashRatio}%`;

        // Positions
        const posTbody = document.querySelector('#inv-positions-table tbody');
        if (snapData.positions && snapData.positions.length > 0) {
          posTbody.innerHTML = snapData.positions.map(pos => `
            <tr>
              <td><strong>${pos.instrument_id.substring(0, 8)}</strong></td>
              <td>${pos.quantity.toLocaleString()}</td>
              <td>$${pos.average_cost.toFixed(1)}</td>
              <td>$${pos.last_known_price.toFixed(1)}</td>
              <td><strong>$${pos.market_value.toLocaleString()}</strong></td>
            </tr>
          `).join('');
        } else {
          posTbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">目前無持倉部位</td></tr>';
        }
      }

      // Constraints
      const consTbody = document.querySelector('#inv-constraints-table tbody');
      if (profData.constraints && profData.constraints.length > 0) {
        consTbody.innerHTML = profData.constraints.map(c => `
          <tr>
            <td><code>${c.constraint_type}</code></td>
            <td><strong>${c.constraint_value}</strong></td>
            <td><span class="badge ${c.enabled ? 'badge-COMPLETED' : 'badge-subtle'}">${c.enabled ? '已啟用' : '停用'}</span></td>
          </tr>
        `).join('');
      }

      // Proposals history
      const props = await this.apiCall('/api/v1/agents/investment/proposals');
      const histTbody = document.querySelector('#inv-proposals-history-table tbody');
      if (props && props.length > 0) {
        histTbody.innerHTML = props.map(p => {
          let d = {};
          try { d = JSON.parse(p.proposal_json); } catch (e) {}
          return `
            <tr>
              <td><strong>${d.symbol || '-'}</strong> ${d.instrument_name || ''}</td>
              <td><span class="badge badge-${p.action}">${p.action}</span></td>
              <td><strong>${p.quantity.toLocaleString()}</strong> 股</td>
              <td>$${p.estimated_price.toFixed(1)}</td>
              <td>$${d.broker_fee || 0}</td>
              <td>$${d.transaction_tax || 0}</td>
              <td><strong class="text-accent">$${p.estimated_total_cost.toLocaleString()}</strong></td>
              <td><span class="badge badge-${p.status}">${p.status}</span></td>
              <td><small class="text-muted">${p.status_reason || d.adjustment_note || '-'}</small></td>
            </tr>
          `;
        }).join('');
      } else {
        histTbody.innerHTML = '<tr><td colspan="9" class="text-center text-muted">無交易提案歷史</td></tr>';
      }

      // Auto-trigger symbol details & documents for currently selected symbol
      const currentSym = document.getElementById('inv-eval-symbol')?.value || '2317';
      await this.onSymbolChange(currentSym);

      // Load Predictions & Calibration
      const preds = await this.apiCall('/api/v1/agents/investment/predictions?limit=10');
      const predTbody = document.querySelector('#inv-predictions-table tbody');
      if (preds && preds.length > 0) {
        predTbody.innerHTML = preds.map(pr => `
          <tr>
            <td><code>${pr.id.substring(0, 8)}</code></td>
            <td>${pr.target}</td>
            <td><span class="badge badge-${pr.expected_direction}">${pr.expected_direction}</span></td>
            <td><strong>${(pr.probability * 100).toFixed(0)}%</strong></td>
            <td>$${pr.base_price.toFixed(1)}</td>
            <td><small>${new Date(pr.due_at).toLocaleDateString()}</small></td>
            <td><span class="badge badge-${pr.status}">${pr.status}</span></td>
          </tr>
        `).join('');
      }

      const evals = await this.apiCall('/api/v1/agents/investment/evaluations?limit=1');
      if (evals && evals.length > 0) {
        const last = evals[0];
        let m = {};
        try { m = JSON.parse(last.metrics_json); } catch(e) {}
        document.getElementById('calib-samples').textContent = last.sample_count;
        if (m.directional_accuracy !== undefined) {
          document.getElementById('calib-accuracy').textContent = (m.directional_accuracy * 100).toFixed(1) + '%';
        }
        if (m.average_brier_score !== undefined) {
          document.getElementById('calib-brier').textContent = m.average_brier_score.toFixed(4);
        }
      }

      // Info & Instructions
      try {
        const info = await this.apiCall('/api/v1/agents/investment/info');
        if (info) {
          const verEl = document.getElementById('inv-agent-version');
          if (verEl) verEl.textContent = `v${info.version || 1}`;
          const instEl = document.getElementById('inv-instructions-input');
          if (instEl && !instEl.value) instEl.value = info.instructions || '';
        }
      } catch (e) {
        console.warn('Investment agent info error:', e);
      }

      // 3. Recent Tasks & History
      await this.loadInvestmentTasks();
      const activeTask = localStorage.getItem('aia_active_task_investment');
      if (activeTask) {
        this.trackTaskProgress('investment', activeTask);
      }
    } catch (e) {
      console.error(e);
    }
  },

  async onSymbolChange(symbol) {
    if (!symbol) return;
    const selEval = document.getElementById('inv-eval-symbol');
    if (selEval && selEval.value !== symbol) selEval.value = symbol;
    const selSync = document.getElementById('inv-sync-symbol');
    if (selSync && selSync.value !== symbol) selSync.value = symbol;

    const stockReasoning = {
      '2317': 'AI 伺服器機櫃拉貨動能強勁，GB200 市佔領先，營收與毛利展望優於預期。',
      '2330': '先進製程 2nm/3nm 產能供不應求，CoWoS 先進封裝持續擴產，維持產業主導地位。',
      '2454': '天璣 9400 旗艦手機晶片出貨動能強勁，邊緣 AI 晶片市佔率擴大。',
      '0050': '追蹤台灣前 50 大龍頭權值股，受惠科技權值股長期獲利增長與穩健配息。'
    };

    if (stockReasoning[symbol] && document.getElementById('inv-eval-reasoning')) {
      document.getElementById('inv-eval-reasoning').value = stockReasoning[symbol];
    }

    try {
      // 1. Fetch latest market data for this symbol
      const data = await this.apiCall(`/api/v1/agents/investment/market-data?symbol=${symbol}&limit=1`);
      if (data && data.length > 0) {
        let b = {};
        try { b = JSON.parse(data[0].value_json); } catch(e) {}
        if (b.close) {
          const priceInput = document.getElementById('inv-eval-price');
          if (priceInput) priceInput.value = b.close;
          const hintEl = document.getElementById('inv-eval-price-hint');
          if (hintEl) hintEl.textContent = `(TWSE 最新收盤價: $${b.close})`;
          const targetInput = document.getElementById('inv-eval-target');
          if (targetInput) targetInput.value = Math.round(b.close * 1000);

          // Update Market Data Summary Card
          const sumBox = document.getElementById('inv-market-data-summary');
          if (sumBox) {
            sumBox.className = 'alert alert-info mb-3';
            sumBox.innerHTML = `
              <strong>📊 ${symbol} 最新盤後行情數據 (TWSE OpenAPI)</strong><br>
              • 收盤價：<strong>$${b.close} TWD</strong> (漲跌：${b.change || 0} / ${b.change_percent ? b.change_percent.toFixed(2) : 0}%)<br>
              • 成交量：${(b.volume || 0).toLocaleString()} 股 | 開高低：${b.open || '-'} / ${b.high || '-'} / ${b.low || '-'}<br>
              • 資料有效時間：${new Date(data[0].effective_at).toLocaleString()}
            `;
            sumBox.classList.remove('hidden');
          }
        }
      }

      // 2. Fetch MOPS documents specifically for this symbol
      await this.loadDocumentsForSymbol(symbol);
    } catch (err) {
      console.warn('Failed to load market data for', symbol, err);
    }
  },

  async loadDocumentsForSymbol(symbol) {
    const docsTbody = document.querySelector('#inv-docs-table tbody');
    if (!docsTbody) return;
    try {
      const docs = await this.apiCall(`/api/v1/agents/investment/documents?symbol=${symbol}&limit=5`);
      if (docs && docs.length > 0) {
        docsTbody.innerHTML = docs.map(d => `
          <tr>
            <td><span class="badge badge-info">${d.content_type}</span></td>
            <td><strong>${this.escape(d.title)}</strong></td>
            <td><small class="text-muted">${this.escape(d.summary || '-')}</small></td>
            <td><small>${new Date(d.published_at).toLocaleDateString()}</small></td>
          </tr>
        `).join('');
      } else {
        docsTbody.innerHTML = `<tr><td colspan="4" class="text-center text-muted">${symbol} 目前無公告紀錄，可點擊「同步最新行情」自動擷取</td></tr>`;
      }
    } catch (e) {
      docsTbody.innerHTML = `<tr><td colspan="4" class="text-center text-muted">載入公告失敗</td></tr>`;
    }
  },

  async handleSyncMarketData() {
    const symbol = document.getElementById('inv-sync-symbol').value || '2330';
    const sumBox = document.getElementById('inv-market-data-summary');
    sumBox.className = 'alert alert-info mb-3';
    sumBox.textContent = `正在自台灣證券交易所 (TWSE) 與觀測站同步 ${symbol} 最新行情...`;
    sumBox.classList.remove('hidden');

    try {
      const res = await this.apiCall('/api/v1/agents/investment/sync', {
        method: 'POST',
        body: { symbol }
      });
      if (res && res.market_data) {
        let b = {};
        try { b = JSON.parse(res.market_data.value_json); } catch(e) {}
        sumBox.className = 'alert alert-success mb-3';
        sumBox.innerHTML = `
          <strong>✓ 成功同步 ${symbol} 最新盤後數據</strong><br>
          • 收盤價：<strong>$${b.close || '-'} TWD</strong> (漲跌：${b.change || 0} / ${b.change_percent ? b.change_percent.toFixed(2) : 0}%)<br>
          • 成交量：${(b.volume || 0).toLocaleString()} 股 | 開高低：${b.open || '-'} / ${b.high || '-'} / ${b.low || '-'}<br>
          • 已同步匯入 ${res.documents_cnt || 0} 則重大訊息與財經公告
        `;
      }
      await this.onSymbolChange(symbol);
    } catch (err) {
      sumBox.className = 'alert alert-danger mb-3';
      sumBox.textContent = '同步失敗: ' + (err.message || err);
    }
  },

  async handleEvaluatePredictions() {
    try {
      const res = await this.apiCall('/api/v1/agents/investment/predictions/evaluate', {
        method: 'POST',
        body: {}
      });
      alert(`已成功結算 ${res.evaluated_outcomes ? res.evaluated_outcomes.length : 0} 筆到期預測，並重新計算校準評分卡！`);
      this.loadInvestment();
    } catch (err) {
      alert('評估結算失敗: ' + (err.message || err));
    }
  },

  async handleEvaluateInvestment() {
    const symbol = document.getElementById('inv-eval-symbol').value;
    const action = document.getElementById('inv-eval-action').value;
    const price = parseFloat(document.getElementById('inv-eval-price').value);
    const target = parseInt(document.getElementById('inv-eval-target').value, 10);
    const conf = parseFloat(document.getElementById('inv-eval-conf').value);
    const horizon = document.getElementById('inv-eval-horizon').value;
    const reasoning = document.getElementById('inv-eval-reasoning').value;

    const resBox = document.getElementById('inv-eval-result');
    resBox.className = 'alert mt-3';

    try {
      const data = await this.apiCall('/api/v1/agents/investment/evaluate', {
        method: 'POST',
        body: {
          symbol,
          action,
          price,
          target_amount: target,
          confidence: conf,
          expected_horizon: horizon,
          reasoning: [reasoning],
          risks: ['Market volatility'],
          strategy_version: 'v1.0'
        }
      });

      const p = data.proposal;
      resBox.className = 'alert alert-success mt-3';
      resBox.innerHTML = `
        <strong>✓ Go 核心驗證成功並生成 Trade Proposal (ID: ${p.id.substring(0, 8)})</strong><br>
        • 方向：<span class="badge badge-${p.action}">${p.action}</span> | 股數：<strong>${p.quantity} 股</strong><br>
        • 預估總成本/款項：<strong>$${p.estimated_total_cost.toLocaleString()} TWD</strong> (已含 Go 計算之手續費及證交稅)<br>
        • 風控說明：${p.status_reason || '符合現有資金上限與風控比率'}
      `;
      this.loadInvestment();
      this.updatePendingCountBadge();
    } catch (err) {
      resBox.className = 'alert alert-error mt-3';
      resBox.innerHTML = `<strong>✗ 投資建議被 Go 核心拒絕：</strong><br>${this.escape(err.message)}`;
    }
  },

  openEditFundsModal() {
    document.getElementById('edit-capital-limit').value = this.currentCapitalLimit != null ? this.currentCapitalLimit : 1000000;
    document.getElementById('edit-available-cash').value = this.currentAvailableCash != null ? this.currentAvailableCash : 600000;
    document.getElementById('edit-funds-modal').classList.remove('hidden');
  },

  closeEditFundsModal() {
    document.getElementById('edit-funds-modal').classList.add('hidden');
  },

  async submitEditFunds(e) {
    e.preventDefault();
    const capLimit = parseInt(document.getElementById('edit-capital-limit').value, 10);
    const availCash = parseInt(document.getElementById('edit-available-cash').value, 10);
    const reason = document.getElementById('edit-funds-reason').value || '手動調整可用現金與資金上限';

    if (isNaN(capLimit) || capLimit < 0 || isNaN(availCash) || availCash < 0) {
      alert('請輸入有效的金額（數值需大於或等於 0）');
      return;
    }

    try {
      await this.apiCall('/api/v1/agents/investment/profile', {
        method: 'POST',
        body: { capital_limit: capLimit }
      });
      await this.apiCall('/api/v1/agents/investment/snapshot', {
        method: 'POST',
        body: { available_cash: availCash, reason: reason }
      });
      this.showToast('✓ 已成功更新可用現金與資金上限！', 'success');
      this.closeEditFundsModal();
      await this.loadInvestment();
    } catch (err) {
      alert('更新失敗: ' + (err.message || err));
    }
  },

  // ----------------------------------------------------------------
  // 5. Memory Tab
  // ----------------------------------------------------------------
  async loadMemory() {
    try {
      const list = await this.apiCall('/api/v1/memory?limit=50');
      const tbody = document.querySelector('#memory-table tbody');
      if (list && list.length > 0) {
        tbody.innerHTML = list.map(m => `
          <tr>
            <td><span class="badge badge-subtle">${m.type}</span></td>
            <td><strong>${this.escape(m.key)}</strong></td>
            <td>${this.escape(m.content)}</td>
            <td>${m.confidence.toFixed(2)}</td>
            <td>${m.credibility.toFixed(2)}</td>
            <td>${this.formatTime(m.created_at)}</td>
          </tr>
        `).join('');
      } else {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">無記憶紀錄</td></tr>';
      }
    } catch (e) {
      console.error(e);
    }
  },

  // ----------------------------------------------------------------
  // 6. Agents Tab
  // ----------------------------------------------------------------
  async loadAgents() {
    try {
      const agents = await this.apiCall('/api/v1/agents');
      const container = document.getElementById('agents-cards-container');
      if (agents && agents.length > 0) {
        container.innerHTML = agents.map(ag => `
          <div class="agent-card">
            <div class="card-header" style="padding: 0 0 0.5rem 0; border: none; background: transparent;">
              <h4>${this.escape(ag.name)}</h4>
              <span class="badge ${ag.enabled ? 'badge-RUNNING' : 'badge-subtle'}">${ag.enabled ? 'ENABLED' : 'DISABLED'}</span>
            </div>
            <p class="text-muted">${this.escape(ag.description)}</p>
            <div class="mt-2"><small class="text-muted">Domain: <code>${ag.domain}</code></small></div>
            <div class="agent-caps">
              ${(ag.capabilities || []).map(c => `<span class="badge badge-subtle">${this.escape(c)}</span>`).join('')}
            </div>
          </div>
        `).join('');
      }
    } catch (e) {
      console.error(e);
    }
  },

  // ----------------------------------------------------------------
  // 7. Audit Tab
  // ----------------------------------------------------------------
  async loadAudit() {
    try {
      const source = document.getElementById('audit-source-filter').value;
      const reqID = document.getElementById('audit-reqid-filter').value.trim();

      let url = '/api/v1/audit?limit=50';
      if (source) url += `&source=${source}`;
      if (reqID) url += `&request_id=${reqID}`;

      const logs = await this.apiCall(url);
      const tbody = document.querySelector('#audit-table tbody');
      if (logs && logs.length > 0) {
        tbody.innerHTML = logs.map(l => {
          let metaStr = '-';
          if (l.metadata && Object.keys(l.metadata).length > 0) {
            metaStr = `<code>${this.escape(JSON.stringify(l.metadata))}</code>`;
          }
          return `
            <tr>
              <td><code>#${l.sequence}</code></td>
              <td><span class="badge badge-subtle">${this.escape(l.source || 'RUNTIME')}</span></td>
              <td><small class="text-muted">${this.escape(l.request_id || '-')}</small></td>
              <td>${this.escape(l.actor_id || l.actor_type)}</td>
              <td><strong>${this.escape(l.action)}</strong></td>
              <td>${this.escape(l.resource_type)}</td>
              <td><span class="badge ${l.outcome === 'SUCCESS' ? 'badge-COMPLETED' : 'badge-FAILED'}">${l.outcome}</span></td>
              <td>${metaStr}</td>
              <td>${this.formatTime(l.created_at)}</td>
            </tr>
          `;
        }).join('');
      } else {
        tbody.innerHTML = '<tr><td colspan="9" class="text-center text-muted">無稽核日誌</td></tr>';
      }
    } catch (e) {
      console.error(e);
    }
  },

  // ----------------------------------------------------------------
  // Investment Prompt & Tasks
  // ----------------------------------------------------------------
  async sendInvestmentPrompt() {
    const input = document.getElementById('inv-prompt-input');
    const status = document.getElementById('inv-prompt-status');
    const prompt = input.value.trim();
    if (!prompt) return this.showToast('請輸入投資研究指令', 'error');

    status.textContent = '⏳ 指派任務中...';
    try {
      const res = await this.apiCall('/api/v1/agents/investment/prompt', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt, async: true })
      });
      status.textContent = '⏳ 任務執行中...';
      localStorage.setItem('aia_active_task_investment', res.task_id);
      input.value = '';
      this.trackTaskProgress('investment', res.task_id);
      this.loadInvestmentTasks();
    } catch (e) {
      status.textContent = '❌ 指派失敗: ' + e.message;
      this.showToast(e.message, 'error');
    }
  },

  async loadInvestmentTasks() {
    try {
      const res = await this.apiCall('/api/v1/agents/investment/tasks?limit=8');
      const tbody = document.querySelector('#inv-tasks-table tbody');
      if (res && res.length > 0) {
        tbody.innerHTML = res.map(t => `
          <tr>
            <td><code>${t.id.substring(0, 8)}</code></td>
            <td><code title="${t.conversation_id || ''}">${t.conversation_id ? t.conversation_id.substring(0, 8) : '-'}</code></td>
            <td>${this.formatTime(t.created_at)}</td>
            <td><strong>${this.escape(t.title)}</strong></td>
            <td><span class="badge badge-${t.status}">${t.status}</span></td>
            <td>
              <button class="btn btn-sm btn-outline" onclick="app.viewTaskProgress('investment', '${t.id}')">🔍 查看日誌/進度</button>
            </td>
          </tr>
        `).join('');
      } else {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">尚無任務歷史</td></tr>';
      }
    } catch (e) {
      console.error(e);
    }
  },

  async saveInvestmentInstructions() {
    const inst = document.getElementById('inv-instructions-input').value;
    const status = document.getElementById('inv-instructions-status');
    status.textContent = '儲存中...';
    try {
      const res = await this.apiCall('/api/v1/agents/investment/instructions', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ instructions: inst })
      });
      status.textContent = '✅ 已儲存 (v' + (res.version || '') + ')';
      const verEl = document.getElementById('inv-agent-version');
      if (verEl && res.version) verEl.textContent = `v${res.version}`;
      this.showToast('Investment Agent 提示詞設定已更新', 'success');
    } catch (e) {
      status.textContent = '❌ 儲存失敗';
      this.showToast('儲存失敗: ' + e.message, 'error');
    }
  },

  // ----------------------------------------------------------------
  // Coding Agent Tab
  // ----------------------------------------------------------------
  async loadCoding() {
    try {
      // 1. Projects
      const projects = await this.apiCall('/api/v1/agents/coding/projects');
      const tbody = document.querySelector('#coding-repos-table tbody');
      const select = document.getElementById('coding-prompt-project');
      select.innerHTML = '';
      if (projects && projects.length > 0) {
        tbody.innerHTML = projects.map(p => `
          <tr>
            <td><code>${this.escape(p.id)}</code></td>
            <td><strong>${this.escape(p.name)}</strong></td>
            <td><span class="badge ${p.status === 'ACTIVE' ? 'badge-COMPLETED' : 'badge-FAILED'}">${p.status}</span></td>
            <td><code>${this.escape(p.workspace)}</code></td>
            <td>
              <button class="btn btn-sm btn-outline" onclick="app.runCodingScan('${this.escape(p.id)}')">🔍 掃描代碼</button>
              <button class="btn btn-sm btn-outline text-danger ml-1" title="從 AIA 解除註冊" onclick="app.deleteCodingRepo('${this.escape(p.id)}', '${this.escape(p.name)}')">🗑️ 移除</button>
            </td>
          </tr>
        `).join('');

        projects.forEach(p => {
          const opt = document.createElement('option');
          opt.value = p.id;
          opt.textContent = `${p.name} (${p.workspace})`;
          select.appendChild(opt);
        });
      } else {
        tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">尚未註冊任何 Coding Repository</td></tr>';
      }

      // 2. Info & Instructions
      const info = await this.apiCall('/api/v1/agents/coding/info');
      if (info) {
        const verEl = document.getElementById('coding-agent-version');
        if (verEl) verEl.textContent = `v${info.version || 1}`;
        const instEl = document.getElementById('coding-instructions-input');
        if (instEl && !instEl.value) instEl.value = info.instructions || '';
      }

      // 3. Recent Tasks & History
      await this.loadCodingTasks();
      const activeTask = localStorage.getItem('aia_active_task_coding');
      if (activeTask) {
        this.trackTaskProgress('coding', activeTask);
      }
    } catch (e) {
      console.error('Failed to load coding agent data:', e);
    }
  },

  openRegisterCodingRepoModal() {
    document.getElementById('register-coding-modal').classList.remove('hidden');
  },

  closeRegisterCodingRepoModal() {
    document.getElementById('register-coding-modal').classList.add('hidden');
  },

  async submitRegisterCodingRepo(event) {
    event.preventDefault();
    const name = document.getElementById('reg-coding-name').value.trim();
    const path = document.getElementById('reg-coding-path').value.trim();
    try {
      await this.apiCall('/api/v1/agents/coding/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, path })
      });
      this.closeRegisterCodingRepoModal();
      this.showToast('Coding Repository 註冊成功！', 'success');
      this.loadCoding();
    } catch (e) {
      this.showToast('註冊失敗: ' + e.message, 'error');
    }
  },

  async deleteCodingRepo(projectID, name) {
    if (!projectID) return;
    if (!confirm(`確定要從 AIA 解除註冊代碼專案「${name || projectID}」嗎？\n\n提示：此操作只會從 AIA 平台移除專案管理與健康度掃描，不會刪除硬碟上的實際代碼庫。`)) {
      return;
    }
    try {
      await this.apiCall('/api/v1/agents/coding/projects?id=' + encodeURIComponent(projectID), {
        method: 'DELETE'
      });
      this.showToast('代碼專案已解除註冊', 'success');
      this.loadCoding();
    } catch (e) {
      this.showToast('移除失敗: ' + e.message, 'error');
    }
  },

  async runCodingScan(projectID) {
    const timestampEl = document.getElementById('coding-scan-timestamp');
    const tbody = document.querySelector('#coding-defects-table tbody');
    tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">掃描中...</td></tr>';
    try {
      const res = await this.apiCall('/api/v1/agents/coding/scan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project_id: projectID || '' })
      });
      timestampEl.textContent = `掃描於: ${new Date().toLocaleTimeString()} (專案: ${res.project_id})`;
      const obs = res.observations || [];
      if (obs.length > 0) {
        tbody.innerHTML = obs.map(o => `
          <tr>
            <td><code>${this.escape(o.category)}</code></td>
            <td><span class="badge ${o.severity >= 8 ? 'badge-FAILED' : (o.severity >= 5 ? 'badge-RUNNING' : 'badge-COMPLETED')}">等級 ${o.severity}</span></td>
            <td>${this.escape(o.summary)}</td>
            <td><code>${this.escape(o.target_agent_id || 'coding-agent')}</code></td>
            <td>${this.formatTime(o.timestamp)}</td>
          </tr>
        `).join('');
        this.showToast(`掃描完成：檢測到 ${obs.length} 項觀測結果`, 'info');
      } else {
        tbody.innerHTML = '<tr><td colspan="5" class="text-center text-accent">✅ 零缺陷！代碼庫與測試套件健康</td></tr>';
        this.showToast('代碼庫完全健康，無缺陷！', 'success');
      }
    } catch (e) {
      tbody.innerHTML = `<tr><td colspan="5" class="text-center text-danger">掃描失敗: ${this.escape(e.message)}</td></tr>`;
      this.showToast('掃描失敗: ' + e.message, 'error');
    }
  },

  async sendCodingPrompt() {
    const projectSelect = document.getElementById('coding-prompt-project');
    const input = document.getElementById('coding-prompt-input');
    const status = document.getElementById('coding-prompt-status');
    const prompt = input.value.trim();
    if (!prompt) return this.showToast('請輸入代碼指令', 'error');

    status.textContent = '⏳ 指派任務中...';
    try {
      const res = await this.apiCall('/api/v1/agents/coding/prompt', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          prompt,
          project_id: projectSelect.value || '',
          async: true
        })
      });
      status.textContent = '⏳ 任務執行中...';
      localStorage.setItem('aia_active_task_coding', res.task_id);
      input.value = '';
      this.trackTaskProgress('coding', res.task_id);
      this.loadCodingTasks();
    } catch (e) {
      status.textContent = '❌ 指派失敗: ' + e.message;
      this.showToast(e.message, 'error');
    }
  },

  async loadCodingTasks() {
    try {
      const res = await this.apiCall('/api/v1/agents/coding/tasks?limit=8');
      const tbody = document.querySelector('#coding-tasks-table tbody');
      if (res && res.length > 0) {
        tbody.innerHTML = res.map(t => `
          <tr>
            <td><code>${t.id.substring(0, 8)}</code></td>
            <td><code title="${t.conversation_id || ''}">${t.conversation_id ? t.conversation_id.substring(0, 8) : '-'}</code></td>
            <td>${this.formatTime(t.created_at)}</td>
            <td><strong>${this.escape(t.title)}</strong></td>
            <td><span class="badge badge-${t.status}">${t.status}</span></td>
            <td>
              <button class="btn btn-sm btn-outline" onclick="app.viewTaskProgress('coding', '${t.id}')">🔍 查看日誌/進度</button>
            </td>
          </tr>
        `).join('');
      } else {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">尚無任務歷史</td></tr>';
      }
    } catch (e) {
      console.error(e);
    }
  },

  async saveCodingInstructions() {
    const inst = document.getElementById('coding-instructions-input').value;
    const status = document.getElementById('coding-instructions-status');
    status.textContent = '儲存中...';
    try {
      const res = await this.apiCall('/api/v1/agents/coding/instructions', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ instructions: inst })
      });
      status.textContent = '✅ 已儲存 (v' + (res.version || '') + ')';
      const verEl = document.getElementById('coding-agent-version');
      if (verEl && res.version) verEl.textContent = `v${res.version}`;
      this.showToast('Coding Agent 提示詞設定已更新', 'success');
    } catch (e) {
      status.textContent = '❌ 儲存失敗';
      this.showToast('儲存失敗: ' + e.message, 'error');
    }
  },

  // ----------------------------------------------------------------
  // Writing Agent Tab
  // ----------------------------------------------------------------
  async loadWriting() {
    try {
      // 1. Projects
      const projects = await this.apiCall('/api/v1/agents/writing/projects');
      const tbody = document.querySelector('#writing-projects-table tbody');
      const select = document.getElementById('writing-prompt-project');
      select.innerHTML = '';
      if (projects && projects.length > 0) {
        tbody.innerHTML = projects.map(p => `
          <tr>
            <td><code>${this.escape(p.id)}</code></td>
            <td><strong>${this.escape(p.name)}</strong></td>
            <td><span class="badge ${p.has_manifest ? 'badge-COMPLETED' : 'badge-FAILED'}">${p.has_manifest ? 'book.yaml' : '無'}</span></td>
            <td><span class="badge ${p.status === 'ACTIVE' ? 'badge-COMPLETED' : 'badge-FAILED'}">${p.status}</span></td>
            <td><code>${this.escape(p.workspace)}</code></td>
            <td>
              <button class="btn btn-sm btn-primary" onclick="app.runWritingScan('${this.escape(p.id)}')">📖 史實校勘</button>
              <button class="btn btn-sm btn-outline ml-1" onclick="app.viewWritingManifest('${this.escape(p.id)}')">📋 檢視 Manifest</button>
              <button class="btn btn-sm btn-outline text-danger ml-1" title="從 AIA 解除註冊" onclick="app.deleteWritingProject('${this.escape(p.id)}', '${this.escape(p.name)}')">🗑️ 移除</button>
            </td>
          </tr>
        `).join('');

        projects.forEach(p => {
          const opt = document.createElement('option');
          opt.value = p.id;
          opt.textContent = `${p.name} (${p.workspace})`;
          select.appendChild(opt);
        });
      } else {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">尚未註冊任何小說寫作專案</td></tr>';
      }

      // 2. Info & Instructions
      const info = await this.apiCall('/api/v1/agents/writing/info');
      if (info) {
        const verEl = document.getElementById('writing-agent-version');
        if (verEl) verEl.textContent = `v${info.version || 1}`;
        const instEl = document.getElementById('writing-instructions-input');
        if (instEl && !instEl.value) instEl.value = info.instructions || '';
      }

      // 3. Recent Tasks & History
      await this.loadWritingTasks();
      const activeTask = localStorage.getItem('aia_active_task_writing');
      if (activeTask) {
        this.trackTaskProgress('writing', activeTask);
      }
    } catch (e) {
      console.error('Failed to load writing agent data:', e);
    }
  },

  openRegisterWritingProjectModal() {
    document.getElementById('register-writing-modal').classList.remove('hidden');
  },

  closeRegisterWritingProjectModal() {
    document.getElementById('register-writing-modal').classList.add('hidden');
  },

  async submitRegisterWritingProject(event) {
    event.preventDefault();
    const name = document.getElementById('reg-writing-name').value.trim();
    const path = document.getElementById('reg-writing-path').value.trim();
    try {
      await this.apiCall('/api/v1/agents/writing/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, path })
      });
      this.closeRegisterWritingProjectModal();
      this.showToast('小說專案註冊成功！', 'success');
      this.loadWriting();
    } catch (e) {
      this.showToast('註冊失敗: ' + e.message, 'error');
    }
  },

  async deleteWritingProject(projectID, name) {
    if (!projectID) return;
    if (!confirm(`確定要從 AIA 解除註冊小說目錄「${name || projectID}」嗎？\n\n提示：此操作只會從 AIA 平台移除專案管理與史實校勘，不會刪除硬碟上的實際小說檔案。`)) {
      return;
    }
    try {
      await this.apiCall('/api/v1/agents/writing/projects?id=' + encodeURIComponent(projectID), {
        method: 'DELETE'
      });
      this.showToast('小說專案已解除註冊', 'success');
      this.loadWriting();
    } catch (e) {
      this.showToast('移除失敗: ' + e.message, 'error');
    }
  },

  async runWritingScan(projectID) {
    const timestampEl = document.getElementById('writing-scan-timestamp');
    const tbody = document.querySelector('#writing-anomalies-table tbody');
    tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">校勘掃描中...</td></tr>';
    try {
      const res = await this.apiCall('/api/v1/agents/writing/scan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project_id: projectID || '' })
      });
      timestampEl.textContent = `校勘於: ${new Date().toLocaleTimeString()} (專案: ${res.project_id})`;
      const obs = res.observations || [];
      if (obs.length > 0) {
        tbody.innerHTML = obs.map(o => `
          <tr>
            <td><code>${this.escape(o.category)}</code></td>
            <td><span class="badge ${o.severity >= 8 ? 'badge-FAILED' : (o.severity >= 5 ? 'badge-RUNNING' : 'badge-COMPLETED')}">等級 ${o.severity}</span></td>
            <td>${this.escape(o.summary)}</td>
            <td><span class="badge badge-subtle">${this.escape(o.data && o.data.knowledge_tier ? o.data.knowledge_tier : '史實/推演')}</span></td>
            <td>${this.formatTime(o.timestamp)}</td>
          </tr>
        `).join('');
        this.showToast(`校勘完成：檢測到 ${obs.length} 項史實或時間線紀錄`, 'info');
      } else {
        tbody.innerHTML = '<tr><td colspan="5" class="text-center text-accent">✅ 全書一致！無年代錯置或人物衝突</td></tr>';
        this.showToast('全書時間線與人物設定一致，無矛盾！', 'success');
      }
    } catch (e) {
      tbody.innerHTML = `<tr><td colspan="5" class="text-center text-danger">校勘失敗: ${this.escape(e.message)}</td></tr>`;
      this.showToast('校勘失敗: ' + e.message, 'error');
    }
  },

  async viewWritingManifest(projectID) {
    const pathEl = document.getElementById('writing-manifest-path');
    const contentEl = document.getElementById('writing-manifest-content');
    contentEl.textContent = '讀取中...';
    try {
      const res = await this.apiCall(`/api/v1/agents/writing/manifest?project_id=${encodeURIComponent(projectID || '')}`);
      pathEl.textContent = res.manifest_file || '';
      contentEl.textContent = JSON.stringify(res.manifest, null, 2);
    } catch (e) {
      pathEl.textContent = '';
      contentEl.textContent = '此小說目錄尚未建立 book.yaml 入口規格。';
      this.showToast('未找到 book.yaml', 'warning');
    }
  },

  async sendWritingPrompt() {
    const projectSelect = document.getElementById('writing-prompt-project');
    const input = document.getElementById('writing-prompt-input');
    const status = document.getElementById('writing-prompt-status');
    const prompt = input.value.trim();
    if (!prompt) return this.showToast('請輸入寫作指令', 'error');

    status.textContent = '⏳ 指派任務中...';
    try {
      const res = await this.apiCall('/api/v1/agents/writing/prompt', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          prompt,
          project_id: projectSelect.value || '',
          async: true
        })
      });
      status.textContent = '⏳ 任務執行中...';
      localStorage.setItem('aia_active_task_writing', res.task_id);
      input.value = '';
      this.trackTaskProgress('writing', res.task_id);
      this.loadWritingTasks();
    } catch (e) {
      status.textContent = '❌ 指派失敗: ' + e.message;
      this.showToast(e.message, 'error');
    }
  },

  async loadWritingTasks() {
    try {
      const res = await this.apiCall('/api/v1/agents/writing/tasks?limit=8');
      const tbody = document.querySelector('#writing-tasks-table tbody');
      if (res && res.length > 0) {
        tbody.innerHTML = res.map(t => `
          <tr>
            <td><code>${t.id.substring(0, 8)}</code></td>
            <td><code title="${t.conversation_id || ''}">${t.conversation_id ? t.conversation_id.substring(0, 8) : '-'}</code></td>
            <td>${this.formatTime(t.created_at)}</td>
            <td><strong>${this.escape(t.title)}</strong></td>
            <td><span class="badge badge-${t.status}">${t.status}</span></td>
            <td>
              <button class="btn btn-sm btn-outline" onclick="app.viewTaskProgress('writing', '${t.id}')">🔍 查看日誌/進度</button>
            </td>
          </tr>
        `).join('');
      } else {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">尚無任務歷史</td></tr>';
      }
    } catch (e) {
      console.error(e);
    }
  },

  async saveWritingInstructions() {
    const inst = document.getElementById('writing-instructions-input').value;
    const status = document.getElementById('writing-instructions-status');
    status.textContent = '儲存中...';
    try {
      const res = await this.apiCall('/api/v1/agents/writing/instructions', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ instructions: inst })
      });
      status.textContent = '✅ 已儲存 (v' + (res.version || '') + ')';
      const verEl = document.getElementById('writing-agent-version');
      if (verEl && res.version) verEl.textContent = `v${res.version}`;
      this.showToast('Writing Agent 提示詞設定已更新', 'success');
    } catch (e) {
      status.textContent = '❌ 儲存失敗';
      this.showToast('儲存失敗: ' + e.message, 'error');
    }
  },

  // ----------------------------------------------------------------
  // Generic Task Progress & Streaming Tracker
  // ----------------------------------------------------------------
  taskPollTimers: {},

  trackTaskProgress(agentID, taskID) {
    if (!taskID) return;
    const prefixMap = {
      'investment': 'inv',
      'investment-agent': 'inv',
      'coding': 'coding',
      'coding-agent': 'coding',
      'writing': 'writing',
      'writing-agent': 'writing'
    };
    const pfx = prefixMap[agentID] || agentID;

    const outBox = document.getElementById(`${pfx}-prompt-output-box`);
    const outPre = document.getElementById(`${pfx}-prompt-output`);
    const taskIdEl = document.getElementById(`${pfx}-prompt-task-id`);
    const badgeEl = document.getElementById(`${pfx}-prompt-badge`);
    const timeEl = document.getElementById(`${pfx}-prompt-time`);
    const statusEl = document.getElementById(`${pfx}-prompt-status`);

    if (outBox) outBox.classList.remove('hidden');
    if (taskIdEl) taskIdEl.textContent = taskID.substring(0, 8);

    if (this.taskPollTimers[agentID]) {
      clearInterval(this.taskPollTimers[agentID]);
      delete this.taskPollTimers[agentID];
    }

    const startTime = Date.now();
    const poll = async () => {
      try {
        const data = await this.apiCall(`/api/v1/tasks/${taskID}/progress`);
        const elapsed = Math.round((Date.now() - startTime) / 1000);

        if (badgeEl) {
          badgeEl.className = `badge badge-${data.status}`;
          badgeEl.textContent = data.status === 'RUNNING' ? '執行中' : (data.status === 'COMPLETED' ? '已完成' : data.status);
        }
        if (timeEl) {
          timeEl.textContent = data.status === 'RUNNING' ? `耗時: ${elapsed}s` : `完成於: ${this.formatTime(data.completed_at || data.created_at)}`;
        }
        if (statusEl) {
          statusEl.textContent = data.status === 'RUNNING' ? '⏳ 背景執行中...' : (data.status === 'COMPLETED' ? '✅ 任務已完成' : `狀態: ${data.status}`);
        }

        const convId = data.conversation_id || (data.execution && data.execution.conversation_id) || '';
        const convIdEl = document.getElementById(`${pfx}-prompt-conv-id`);
        if (convIdEl) {
          convIdEl.textContent = convId ? convId.substring(0, 8) : '-';
          convIdEl.title = convId || '等待 agy 對話 ID 分配中...';
        }

        let displayText = `【任務 ID】: ${data.task_id}\n【任務標題】: ${data.title}\n【當前狀態】: ${data.status}`;
        if (convId) {
          displayText += `\n【agy 對話 ID】: ${convId}`;
        }
        if (data.execution) {
          displayText += `\n【執行 ID】: ${data.execution.id || ''}  |  【Result】: ${data.execution.result_status || '-'}  |  【Tokens】: ${data.execution.total_tokens || 0}`;
          if (data.execution.error_message) displayText += `\n【Error】: ${data.execution.error_message}`;
        }
        if (data.output && data.output.trim() !== '') {
          displayText += `\n\n--- 執行輸出與觀測日誌 ---\n${data.output}`;
        } else if (data.status === 'RUNNING') {
          displayText += `\n\n⏳ 任務正在沙盒工作區執行中，請稍候... (已運行 ${elapsed} 秒)`;
        }
        if (outPre) {
          outPre.textContent = displayText;
          outPre.scrollTop = outPre.scrollHeight;
        }

        if (data.status === 'COMPLETED' || data.status === 'FAILED' || data.status === 'CANCELLED') {
          clearInterval(this.taskPollTimers[agentID]);
          delete this.taskPollTimers[agentID];
          localStorage.removeItem('aia_active_task_' + agentID);
          if (data.status === 'COMPLETED') {
            this.showToast(`任務 (${taskID.substring(0, 8)}) 已成功完成！`, 'success');
          } else {
            this.showToast(`任務 (${taskID.substring(0, 8)}) 結束: ${data.status}`, 'warning');
          }
          if (pfx === 'coding') this.loadCodingTasks();
          else if (pfx === 'writing') this.loadWritingTasks();
          else if (pfx === 'inv') this.loadInvestmentTasks();
        }
      } catch (e) {
        console.error('Progress poll error:', e);
      }
    };

    poll();
    this.taskPollTimers[agentID] = setInterval(poll, 1500);
  },

  async viewTaskProgress(agentID, taskID) {
    const prefixMap = { 'investment': 'inv', 'coding': 'coding', 'writing': 'writing' };
    const pfx = prefixMap[agentID] || agentID;
    const box = document.getElementById(`${pfx}-prompt-output-box`);
    if (box) {
      box.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
    this.trackTaskProgress(agentID, taskID);
  },

  // Helpers
  escape(str) {
    if (!str) return '';
    return String(str).replace(/[&<>"']/g, s => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[s]));
  },

  formatTime(isoStr) {
    if (!isoStr) return '-';
    try {
      const d = new Date(isoStr);
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    } catch (e) {
      return isoStr;
    }
  },

  showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    container.appendChild(toast);
    setTimeout(() => toast.remove(), 4000);
  }
};

window.addEventListener('DOMContentLoaded', () => {
  app.init();
});
