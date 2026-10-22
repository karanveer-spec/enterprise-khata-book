// KhataBook - Modern Fintech Application Logic
(function () {
  'use strict';

  // Global App State
  const state = {
    currentUserId: localStorage.getItem('khata_user_id') || 'admin',
    currentTheme: localStorage.getItem('khata_theme') || 'light',
    currentView: 'ledger', // 'ledger' (Department Hierarchy Tree & Ledger) or 'overview'
    users: [],
    departments: [],
    activeDeptId: null,
    activeTree: null,
    activeNodeId: null,
    activeNodeDetails: null,
    entriesViewMode: 'direct', // 'direct' or 'consolidated'
    collapsedNodes: new Set(),

    // Table Data & Controls State
    rawEntries: [],
    filteredEntries: [],
    selectedEntryIds: new Set(),
    sortColumn: 'entry_date',
    sortDirection: 'desc',
    searchQuery: '',
    dateFilter: 'all', // 'all', 'month', 'quarter', 'year', 'custom'
    dateStart: '',
    dateEnd: '',
    typeFilter: 'all', // 'all', 'CREDIT', 'DEBIT'
    currentPage: 1,
    pageSize: 10,
  };

  const MAX_DEPTH = 5;

  // DOM Elements
  const el = {
    // Theme & Navigation
    themeToggleBtn: document.getElementById('themeToggleBtn'),
    themeIcon: document.getElementById('themeIcon'),
    navViewOverview: document.getElementById('navViewOverview'),
    navViewLedger: document.getElementById('navViewLedger'),
    navCenterControls: document.getElementById('navCenterControls'),
    globalOverviewView: document.getElementById('globalOverviewView'),
    departmentLedgerView: document.getElementById('departmentLedgerView'),

    // Global Overview
    ovTotalDepts: document.getElementById('ovTotalDepts'),
    ovTotalNodes: document.getElementById('ovTotalNodes'),
    ovTotalCredits: document.getElementById('ovTotalCredits'),
    ovTotalDebits: document.getElementById('ovTotalDebits'),
    ovNetBalance: document.getElementById('ovNetBalance'),
    ovBalanceBadge: document.getElementById('ovBalanceBadge'),
    overviewDeptsGrid: document.getElementById('overviewDeptsGrid'),
    btnOverviewNewDept: document.getElementById('btnOverviewNewDept'),

    // Top Bar
    userSelect: document.getElementById('userSelect'),
    userBadge: document.getElementById('userBadge'),
    userAvatar: document.getElementById('userAvatar'),
    deptSelect: document.getElementById('deptSelect'),
    btnNewDept: document.getElementById('btnNewDept'),

    // Sidebar & Tree
    treeContainer: document.getElementById('treeContainer'),
    breadcrumbsList: document.getElementById('breadcrumbsList'),
    nodeTitle: document.getElementById('nodeTitle'),
    nodeDepthBadge: document.getElementById('nodeDepthBadge'),
    nodeActions: document.getElementById('nodeActions'),
    btnAddSubgroup: document.getElementById('btnAddSubgroup'),
    btnMoveNode: document.getElementById('btnMoveNode'),
    btnRenameNode: document.getElementById('btnRenameNode'),
    btnDeleteNode: document.getElementById('btnDeleteNode'),
    maxDepthBanner: document.getElementById('maxDepthBanner'),

    // Rollup Metrics & Visual Trends
    metricTotalCredits: document.getElementById('metricTotalCredits'),
    metricDirectCredits: document.getElementById('metricDirectCredits'),
    metricDescCredits: document.getElementById('metricDescCredits'),
    metricTotalDebits: document.getElementById('metricTotalDebits'),
    metricDirectDebits: document.getElementById('metricDirectDebits'),
    metricDescDebits: document.getElementById('metricDescDebits'),
    metricNetBalance: document.getElementById('metricNetBalance'),
    balanceStatusChip: document.getElementById('balanceStatusChip'),
    metricTotalCount: document.getElementById('metricTotalCount'),
    metricDirectCount: document.getElementById('metricDirectCount'),
    trendsCard: document.getElementById('trendsCard'),
    trendChartContainer: document.getElementById('trendChartContainer'),

    // Entries Section & Toolbar
    tabDirect: document.getElementById('tabDirectEntries'),
    tabConsolidated: document.getElementById('tabConsolidatedEntries'),
    directCountBadge: document.getElementById('directCountBadge'),
    consolidatedCountBadge: document.getElementById('consolidatedCountBadge'),
    btnNewEntry: document.getElementById('btnNewEntry'),
    btnExportCSV: document.getElementById('btnExportCSV'),
    btnPrintPDF: document.getElementById('btnPrintPDF'),

    // Table Filtering
    tableSearchInput: document.getElementById('tableSearchInput'),
    btnClearSearch: document.getElementById('btnClearSearch'),
    customDateRangeBox: document.getElementById('customDateRangeBox'),
    dateFilterStart: document.getElementById('dateFilterStart'),
    dateFilterEnd: document.getElementById('dateFilterEnd'),

    // Bulk Action Bar
    bulkActionBar: document.getElementById('bulkActionBar'),
    bulkSelectedCount: document.getElementById('bulkSelectedCount'),
    btnBulkMove: document.getElementById('btnBulkMove'),
    btnBulkExport: document.getElementById('btnBulkExport'),
    btnBulkDelete: document.getElementById('btnBulkDelete'),
    btnCancelBulk: document.getElementById('btnCancelBulk'),

    // Table & Pagination
    selectAllCheckbox: document.getElementById('selectAllCheckbox'),
    entriesTable: document.getElementById('entriesTable'),
    entriesTableBody: document.getElementById('entriesTableBody'),
    thOriginNode: document.getElementById('thOriginNode'),
    pageSizeSelect: document.getElementById('pageSizeSelect'),
    paginationInfo: document.getElementById('paginationInfo'),
    paginationCurrentPage: document.getElementById('paginationCurrentPage'),
    btnPageFirst: document.getElementById('btnPageFirst'),
    btnPagePrev: document.getElementById('btnPagePrev'),
    btnPageNext: document.getElementById('btnPageNext'),
    btnPageLast: document.getElementById('btnPageLast'),

    // Modals
    createNodeModal: document.getElementById('createNodeModal'),
    createNodeForm: document.getElementById('createNodeForm'),
    modalNodeTitle: document.getElementById('modalNodeTitle'),
    modalParentName: document.getElementById('modalParentName'),
    modalNewDepthBadge: document.getElementById('modalNewDepthBadge'),
    nodeNameInput: document.getElementById('nodeNameInput'),

    createEntryModal: document.getElementById('createEntryModal'),
    createEntryForm: document.getElementById('createEntryForm'),
    entryAmount: document.getElementById('entryAmount'),
    entryDate: document.getElementById('entryDate'),
    entryReference: document.getElementById('entryReference'),
    entryDescription: document.getElementById('entryDescription'),

    moveNodeModal: document.getElementById('moveNodeModal'),
    moveNodeForm: document.getElementById('moveNodeForm'),
    moveNodeTargetName: document.getElementById('moveNodeTargetName'),
    moveNewParentSelect: document.getElementById('moveNewParentSelect'),
    moveTargetDepth: document.getElementById('moveTargetDepth'),
    moveSubtreeHeight: document.getElementById('moveSubtreeHeight'),

    renameNodeModal: document.getElementById('renameNodeModal'),
    renameNodeForm: document.getElementById('renameNodeForm'),
    renameNodeInput: document.getElementById('renameNodeInput'),

    bulkMoveModal: document.getElementById('bulkMoveModal'),
    bulkMoveForm: document.getElementById('bulkMoveForm'),
    bulkMoveCountLabel: document.getElementById('bulkMoveCountLabel'),
    bulkMoveTargetSelect: document.getElementById('bulkMoveTargetSelect'),

    // Official Report Statement Elements
    reportModal: document.getElementById('reportModal'),
    modalReportPreviewTarget: document.getElementById('modalReportPreviewTarget'),
    btnTriggerPrint: document.getElementById('btnTriggerPrint'),
    btnModalPrintDoc: document.getElementById('btnModalPrintDoc'),
    scopeConsolidated: document.getElementById('scopeConsolidated'),
    scopeDirect: document.getElementById('scopeDirect'),
    officialReportContainer: document.getElementById('officialReportContainer'),

    // Bento Dashboard Elements
    sidebarNavOverview: document.getElementById('sidebarNavOverview'),
    sidebarNavLedger: document.getElementById('sidebarNavLedger'),
    sidebarNavReport: document.getElementById('sidebarNavReport'),
    sidebarNavUsers: document.getElementById('sidebarNavUsers'),
    sidebarHelpBtn: document.getElementById('sidebarHelpBtn'),
    bentoGreetingUser: document.getElementById('bentoGreetingUser'),
    teamAvatarsStack: document.getElementById('teamAvatarsStack'),
    smartcardUser: document.getElementById('smartcardUser'),
    bentoRecentTxList: document.getElementById('bentoRecentTxList'),
    btnSwitchToFullLedger: document.getElementById('btnSwitchToFullLedger'),
    bentoExpensesPercent: document.getElementById('bentoExpensesPercent'),
    bentoIncomePercent: document.getElementById('bentoIncomePercent'),
    bentoBarIncome: document.getElementById('bentoBarIncome'),
    bentoBarExpenses: document.getElementById('bentoBarExpenses'),
    gaugePercentage: document.getElementById('gaugePercentage'),
    gaugeProgressArc: document.getElementById('gaugeProgressArc'),
    btnBentoOpenReport: document.getElementById('btnBentoOpenReport'),
    topNavSearchInput: document.getElementById('topNavSearchInput'),

    toastContainer: document.getElementById('toastContainer'),
  };

  // Helper: Authenticated API Fetch
  async function apiFetch(url, options = {}) {
    const headers = {
      'Content-Type': 'application/json',
      'X-User-Id': state.currentUserId || 'admin',
      ...(options.headers || {}),
    };

    try {
      const res = await fetch(url, { ...options, headers });
      const data = await res.json().catch(() => ({}));

      if (!res.ok) {
        const errorMsg = data.error || `HTTP error ${res.status}`;
        showToast(errorMsg, 'error');
        throw new Error(errorMsg);
      }
      return data;
    } catch (err) {
      console.error('API Error:', err);
      throw err;
    }
  }

  // Toast Notification
  function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast-item toast-${type}`;
    toast.textContent = message;
    el.toastContainer.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateY(10px)';
      toast.style.transition = 'all 0.25s ease';
      setTimeout(() => toast.remove(), 250);
    }, 3200);
  }

  // Currency Formatter - Indian Rupees (₹) with Indian Lakhs/Crores numbering
  function formatMoney(amount) {
    if (isNaN(amount) || amount === null || amount === undefined) {
      amount = 0;
    }
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency: 'INR',
      maximumFractionDigits: 2,
    }).format(amount);
  }

  // Init App
  async function init() {
    initTheme();
    setupModalListeners();
    setupEventListeners();

    try {
      await loadUsers();
    } catch (err) {
      console.warn('Failed loading users list:', err);
    }
    await loadDepartments();
    setView(state.currentView);
  }

  // Theme Management (Dark / Light Mode)
  function initTheme() {
    applyTheme(state.currentTheme);
    el.themeToggleBtn.onclick = () => {
      const newTheme = state.currentTheme === 'dark' ? 'light' : 'dark';
      state.currentTheme = newTheme;
      localStorage.setItem('khata_theme', newTheme);
      applyTheme(newTheme);
      showToast(`Switched to ${newTheme} mode`, 'info');
    };
  }

  function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    el.themeIcon.textContent = theme === 'dark' ? '☀️' : '🌙';
  }

  // View Navigation: Overview vs Ledger
  function setView(viewName) {
    state.currentView = viewName;
    if (viewName === 'overview') {
      if (el.sidebarNavOverview) el.sidebarNavOverview.classList.add('active');
      if (el.sidebarNavLedger) el.sidebarNavLedger.classList.remove('active');
      if (el.navViewOverview) el.navViewOverview.classList.add('active');
      if (el.navViewLedger) el.navViewLedger.classList.remove('active');
      el.globalOverviewView.classList.remove('hidden');
      el.departmentLedgerView.classList.add('hidden');
      el.navCenterControls.classList.add('hidden');
      loadGlobalOverview();
    } else {
      if (el.sidebarNavLedger) el.sidebarNavLedger.classList.add('active');
      if (el.sidebarNavOverview) el.sidebarNavOverview.classList.remove('active');
      if (el.navViewLedger) el.navViewLedger.classList.add('active');
      if (el.navViewOverview) el.navViewOverview.classList.remove('active');
      el.departmentLedgerView.classList.remove('hidden');
      el.globalOverviewView.classList.add('hidden');
      el.navCenterControls.classList.remove('hidden');
    }
  }

  // Render clickable team avatars stack in Bento header
  function renderTeamAvatars() {
    if (!el.teamAvatarsStack) return;
    if (!state.users || state.users.length === 0) return;

    el.teamAvatarsStack.innerHTML = state.users.map(u => {
      const isCurrent = u.id === state.currentUserId;
      const initial = u.name ? u.name.charAt(0).toUpperCase() : 'U';
      return `<div class="avatar-bubble ${isCurrent ? 'active-bubble' : ''}" data-user-id="${u.id}" title="${escapeHTML(u.name)} (${escapeHTML(u.role)})">${initial}</div>`;
    }).join('') + `<div class="avatar-bubble" style="background: transparent; border: none; font-size: 16px; color: #94a3b8;">›</div>`;

    el.teamAvatarsStack.querySelectorAll('.avatar-bubble[data-user-id]').forEach(b => {
      b.onclick = async () => {
        const uid = b.dataset.userId;
        if (state.currentUserId === uid) return;
        state.currentUserId = uid;
        localStorage.setItem('khata_user_id', uid);
        el.userSelect.value = uid;
        updateUserBadge();
        showToast(`Switched user context to: ${uNameById(uid)}`, 'info');
        state.activeDeptId = null;
        state.activeNodeId = null;
        if (state.currentView === 'overview') {
          await loadGlobalOverview();
        } else {
          await loadDepartments();
        }
      };
    });
  }

  function uNameById(id) {
    const u = state.users.find(x => x.id === id);
    return u ? u.name : id;
  }

  // Load Global Overview
  async function loadGlobalOverview() {
    try {
      const data = await apiFetch('/api/overview');
      if (!data) return;

      // Update Active User in Greeting & Virtual Smartcard
      const activeUser = state.users.find(u => u.id === state.currentUserId);
      if (activeUser) {
        if (el.bentoGreetingUser) el.bentoGreetingUser.textContent = activeUser.name;
        if (el.smartcardUser) el.smartcardUser.textContent = activeUser.name;
      }
      renderTeamAvatars();

      el.ovTotalDepts.textContent = data.total_departments;
      el.ovTotalNodes.textContent = `${data.total_nodes} total hierarchy nodes`;
      el.ovTotalCredits.textContent = formatMoney(data.total_credits);
      el.ovTotalDebits.textContent = formatMoney(data.total_debits);
      el.ovNetBalance.textContent = (data.net_balance >= 0 ? '+' : '') + formatMoney(data.net_balance);

      if (data.net_balance > 0) {
        el.ovBalanceBadge.textContent = 'Net Surplus (Jama)';
        el.ovBalanceBadge.className = 'bento-pill-badge';
      } else if (data.net_balance < 0) {
        el.ovBalanceBadge.textContent = 'Net Deficit (Udhar)';
        el.ovBalanceBadge.className = 'bento-pill-badge';
        el.ovBalanceBadge.style.background = '#fef2f2';
        el.ovBalanceBadge.style.color = '#dc2626';
      } else {
        el.ovBalanceBadge.textContent = 'Balanced';
        el.ovBalanceBadge.className = 'bento-pill-badge';
      }

      // Update Expenses & Income Ratio Card
      const totalVolume = (data.total_credits || 0) + (data.total_debits || 0);
      let incomeRatio = 55;
      let expensesRatio = 45;
      if (totalVolume > 0) {
        incomeRatio = Math.round((data.total_credits / totalVolume) * 100);
        expensesRatio = 100 - incomeRatio;
      }
      if (el.bentoIncomePercent) el.bentoIncomePercent.textContent = `${incomeRatio}%`;
      if (el.bentoExpensesPercent) el.bentoExpensesPercent.textContent = `${expensesRatio}%`;
      if (el.bentoBarIncome) el.bentoBarIncome.style.width = `${incomeRatio}%`;
      if (el.bentoBarExpenses) el.bentoBarExpenses.style.width = `${expensesRatio}%`;

      // Update Radial Gauge Arc
      if (el.gaugePercentage) el.gaugePercentage.textContent = `${incomeRatio}%`;
      if (el.gaugeProgressArc) {
        const maxOffset = 126;
        const offset = Math.max(0, Math.min(126, maxOffset - (maxOffset * (incomeRatio / 100))));
        el.gaugeProgressArc.style.strokeDashoffset = offset;
      }

      // Populate Recent Transactions List in Bento Card 4
      if (el.bentoRecentTxList && data.departments && data.departments.length > 0) {
        try {
          const firstDept = data.departments[0];
          const recentsRes = await apiFetch(`/api/nodes/${firstDept.root_node_id}/entries?include_descendants=true&limit=4`);
          const recents = Array.isArray(recentsRes) ? recentsRes : (recentsRes && recentsRes.items ? recentsRes.items : []);
          if (recents.length > 0) {
            el.bentoRecentTxList.innerHTML = recents.map(r => {
              const isCredit = r.type === 'CREDIT';
              return `
                <div class="tx-item">
                  <div class="tx-icon-circle ${isCredit ? 'bg-blue-soft' : 'bg-red-soft'}">${isCredit ? '📥' : '📤'}</div>
                  <div class="tx-info">
                    <strong class="tx-title">${escapeHTML(r.description)}</strong>
                    <span class="tx-date">${escapeHTML(r.entry_date)} • ${escapeHTML(r.reference || 'Regular')}</span>
                  </div>
                  <div class="tx-amount ${isCredit ? 'text-jama' : 'text-udhar'}">${isCredit ? '+' : '-'}${formatMoney(r.amount)}</div>
                </div>
              `;
            }).join('');
          }
        } catch (err) {}
      }

      // Render Department Cards
      if (!data.departments || data.departments.length === 0) {
        el.overviewDeptsGrid.innerHTML = '<div class="empty-state">No accessible departments.</div>';
        return;
      }

      el.overviewDeptsGrid.innerHTML = data.departments.map(d => {
        const net = d.rollup ? d.rollup.total_balance : 0;
        const credits = d.rollup ? d.rollup.total_credits : 0;
        const debits = d.rollup ? d.rollup.total_debits : 0;

        return `
          <div class="dept-card-box">
            <div class="dept-box-header">
              <div>
                <h4 class="dept-box-name">${escapeHTML(d.name)}</h4>
                <div class="dept-box-caption">${d.nodes_count} groups & subgroups | Max Depth: ${MAX_DEPTH}</div>
              </div>
              <span class="level-indicator level-1">L1 Dept</span>
            </div>

            <div class="dept-box-metrics">
              <div class="dept-mini-metric">
                <span class="dept-mini-label">Total Jama</span>
                <span class="dept-mini-amount text-jama">${formatMoney(credits)}</span>
              </div>
              <div class="dept-mini-metric">
                <span class="dept-mini-label">Total Udhar</span>
                <span class="dept-mini-amount text-udhar">${formatMoney(debits)}</span>
              </div>
            </div>

            <div class="dept-box-footer">
              <span class="status-chip ${net >= 0 ? 'chip-surplus' : 'chip-deficit'}">
                Net: ${net >= 0 ? '+' : ''}${formatMoney(net)}
              </span>
              <button class="btn btn-sm btn-outline" data-open-dept="${d.id}">
                Open Ledger ➔
              </button>
            </div>
          </div>
        `;
      }).join('');

      // Bind Open Dept buttons
      el.overviewDeptsGrid.querySelectorAll('[data-open-dept]').forEach(btn => {
        btn.onclick = async () => {
          state.activeDeptId = btn.dataset.openDept;
          el.deptSelect.value = state.activeDeptId;
          setView('ledger');
          await loadDepartmentTree(state.activeDeptId);
        };
      });

    } catch (err) {
      console.error('Failed to load global overview', err);
    }
  }

  // Load Users & Setup User Switcher
  async function loadUsers() {
    const users = await apiFetch('/api/users');
    state.users = users || [];
    el.userSelect.innerHTML = state.users
      .map(u => `<option value="${u.id}" ${u.id === state.currentUserId ? 'selected' : ''}>${u.name}</option>`)
      .join('');

    updateUserBadge();
    renderTeamAvatars();
    const activeUser = state.users.find(u => u.id === state.currentUserId);
    if (activeUser) {
      if (el.bentoGreetingUser) el.bentoGreetingUser.textContent = activeUser.name;
      if (el.smartcardUser) el.smartcardUser.textContent = activeUser.name;
    }
  }

  function updateUserBadge() {
    const u = state.users.find(x => x.id === state.currentUserId);
    if (!u) return;

    if (el.userAvatar) {
      el.userAvatar.textContent = u.name.charAt(0).toUpperCase();
    }

    if (u.role === 'admin') {
      el.userBadge.textContent = 'Admin';
      el.userBadge.className = 'role-pill';
      el.btnNewDept.classList.remove('hidden');
      if (el.btnOverviewNewDept) el.btnOverviewNewDept.classList.remove('hidden');
    } else {
      el.userBadge.textContent = 'Scoped';
      el.userBadge.className = 'role-pill';
      el.userBadge.style.backgroundColor = 'var(--depth-3)';
      el.btnNewDept.classList.add('hidden');
      if (el.btnOverviewNewDept) el.btnOverviewNewDept.classList.add('hidden');
    }
  }

  // Load Departments for Active User
  async function loadDepartments() {
    el.treeContainer.innerHTML = '<div class="loading-state">Loading hierarchy...</div>';
    try {
      const depts = await apiFetch('/api/departments');
      state.departments = depts || [];

      if (state.departments.length === 0) {
        el.deptSelect.innerHTML = '<option value="">No Accessible Departments</option>';
        el.treeContainer.innerHTML = '<div class="loading-state">No departments assigned to your account.</div>';
        clearNodeView();
        return;
      }

      el.deptSelect.innerHTML = state.departments
        .map(d => `<option value="${d.id}" ${d.id === state.activeDeptId ? 'selected' : ''}>${d.name}</option>`)
        .join('');

      if (!state.activeDeptId || !state.departments.some(d => d.id === state.activeDeptId)) {
        state.activeDeptId = state.departments[0].id;
      }

      el.deptSelect.value = state.activeDeptId;
      await loadDepartmentTree(state.activeDeptId);
    } catch (err) {
      console.error('Failed to load departments', err);
      el.treeContainer.innerHTML = `<div class="loading-state">Error loading departments: ${err.message}</div>`;
      clearNodeView();
    }
  }

  // Load Tree for Selected Department
  async function loadDepartmentTree(deptId) {
    el.treeContainer.innerHTML = '<div class="loading-state">Loading hierarchy...</div>';
    try {
      const tree = await apiFetch(`/api/departments/${deptId}/tree`);
      state.activeTree = tree;

      renderTree(tree);

      const targetNodeId = state.activeNodeId && findNodeInTree(tree, state.activeNodeId)
        ? state.activeNodeId
        : tree.id;

      await selectNode(targetNodeId);
    } catch (err) {
      console.error('Failed to load department tree', err);
      el.treeContainer.innerHTML = `<div class="loading-state">Access denied or error loading department tree.</div>`;
      clearNodeView();
    }
  }

  function findNodeInTree(node, id) {
    if (!node) return null;
    if (node.id === id) return node;
    if (node.children) {
      for (const child of node.children) {
        const found = findNodeInTree(child, id);
        if (found) return found;
      }
    }
    return null;
  }

  function collectFlatNodes(node, list = []) {
    if (!node) return list;
    list.push(node);
    if (node.children) {
      for (const child of node.children) {
        collectFlatNodes(child, list);
      }
    }
    return list;
  }

  function getSubtreeMaxDepth(node) {
    if (!node) return 0;
    let max = node.depth;
    if (node.children) {
      for (const child of node.children) {
        max = Math.max(max, getSubtreeMaxDepth(child));
      }
    }
    return max;
  }

  // Render Sidebar Tree
  function renderTree(rootNode) {
    el.treeContainer.innerHTML = '';
    if (!rootNode) {
      el.treeContainer.innerHTML = '<div class="loading-state">No hierarchy found.</div>';
      return;
    }
    const rootEl = createTreeNodeElement(rootNode);
    el.treeContainer.appendChild(rootEl);
  }

  function createTreeNodeElement(node) {
    const container = document.createElement('div');
    container.className = 'tree-node-item';
    container.dataset.nodeId = node.id;

    const hasChildren = node.children && node.children.length > 0;
    const isCollapsed = state.collapsedNodes.has(node.id);

    const row = document.createElement('div');
    row.className = `tree-node-row ${node.id === state.activeNodeId ? 'active' : ''}`;
    row.dataset.id = node.id;
    row.title = `${node.name} (Depth ${node.depth})`;

    const expander = document.createElement('span');
    expander.className = `tree-expander ${hasChildren ? (isCollapsed ? 'collapsed' : '') : 'empty'}`;
    expander.innerHTML = '▼';
    expander.onclick = (e) => {
      e.stopPropagation();
      if (!hasChildren) return;
      if (state.collapsedNodes.has(node.id)) {
        state.collapsedNodes.delete(node.id);
      } else {
        state.collapsedNodes.add(node.id);
      }
      renderTree(state.activeTree);
    };
    row.appendChild(expander);

    // Folder icon
    const icon = document.createElement('span');
    icon.className = 'tree-icon';
    icon.textContent = node.depth === 1 ? '🏢' : (hasChildren ? (isCollapsed ? '📁' : '📂') : '📄');
    row.appendChild(icon);

    const depthPill = document.createElement('span');
    depthPill.className = `tree-depth-pill`;
    depthPill.textContent = `L${node.depth}`;
    row.appendChild(depthPill);

    const title = document.createElement('span');
    title.className = 'tree-node-title';
    title.textContent = node.name;
    row.appendChild(title);

    if (node.rollup) {
      const bal = node.rollup.total_balance;
      const balPill = document.createElement('span');
      balPill.className = `tree-balance-pill ${bal >= 0 ? 'positive' : 'negative'}`;
      balPill.textContent = (bal >= 0 ? '+' : '') + formatMoney(bal);
      row.appendChild(balPill);
    }

    row.onclick = () => selectNode(node.id);
    container.appendChild(row);

    if (hasChildren && !isCollapsed) {
      const childrenBox = document.createElement('div');
      childrenBox.className = 'tree-children';
      for (const child of node.children) {
        childrenBox.appendChild(createTreeNodeElement(child));
      }
      container.appendChild(childrenBox);
    }

    return container;
  }

  // Select Node
  async function selectNode(nodeId) {
    state.activeNodeId = nodeId;
    state.selectedEntryIds.clear();
    updateBulkActionBar();

    document.querySelectorAll('.tree-node-row').forEach(r => {
      r.classList.toggle('active', r.dataset.id === nodeId);
    });

    try {
      const data = await apiFetch(`/api/nodes/${nodeId}`);
      state.activeNodeDetails = data;

      renderNodeDetails(data);
      await loadEntries();
    } catch (err) {
      console.error('Failed to load node details', err);
    }
  }

  function clearNodeView() {
    el.nodeTitle.textContent = 'No Node Selected';
    el.breadcrumbsList.innerHTML = '<li>--</li>';
    el.metricTotalCredits.textContent = '₹0.00';
    el.metricTotalDebits.textContent = '₹0.00';
    el.metricNetBalance.textContent = '₹0.00';
    el.entriesTableBody.innerHTML = '<tr><td colspan="8" class="empty-state">No node selected.</td></tr>';
    renderVisualTrendChart([]);
  }

  function renderNodeDetails(data) {
    const { node, ancestors, rollup } = data;

    // Breadcrumbs
    el.breadcrumbsList.innerHTML = ancestors
      .map((a, idx) => {
        const isCurrent = a.id === node.id;
        if (isCurrent) {
          return `<li class="current-crumb"><strong>${escapeHTML(a.name)}</strong> (L${a.depth})</li>`;
        }
        return `<li><a data-crumb-id="${a.id}">${escapeHTML(a.name)}</a></li>`;
      })
      .join('');

    el.breadcrumbsList.querySelectorAll('a[data-crumb-id]').forEach(link => {
      link.onclick = () => selectNode(link.dataset.crumbId);
    });

    // Node Header
    el.nodeTitle.textContent = node.name;
    el.nodeDepthBadge.textContent = `Depth ${node.depth} of ${MAX_DEPTH} (${getDepthLabel(node.depth)})`;
    el.nodeDepthBadge.className = `level-indicator level-${node.depth}`;

    const isMaxDepth = node.depth >= MAX_DEPTH;
    el.btnAddSubgroup.disabled = isMaxDepth;
    if (isMaxDepth) {
      el.btnAddSubgroup.title = 'Cannot add subgroup: Maximum depth limit of 5 is reached';
      el.maxDepthBanner.classList.remove('hidden');
    } else {
      el.btnAddSubgroup.title = 'Add child subgroup under this node';
      el.maxDepthBanner.classList.add('hidden');
    }

    el.btnMoveNode.style.display = node.depth === 1 ? 'none' : 'inline-flex';

    // Metrics
    if (rollup) {
      el.metricTotalCredits.textContent = formatMoney(rollup.total_credits);
      el.metricDirectCredits.textContent = formatMoney(rollup.direct_credits);
      el.metricDescCredits.textContent = formatMoney(rollup.total_credits - rollup.direct_credits);

      el.metricTotalDebits.textContent = formatMoney(rollup.total_debits);
      el.metricDirectDebits.textContent = formatMoney(rollup.direct_debits);
      el.metricDescDebits.textContent = formatMoney(rollup.total_debits - rollup.direct_debits);

      const net = rollup.total_balance;
      el.metricNetBalance.textContent = (net >= 0 ? '+' : '') + formatMoney(net);

      if (net > 0) {
        el.balanceStatusChip.textContent = 'Surplus (Net Jama)';
        el.balanceStatusChip.className = 'status-chip chip-surplus';
      } else if (net < 0) {
        el.balanceStatusChip.textContent = 'Deficit (Net Udhar)';
        el.balanceStatusChip.className = 'status-chip chip-deficit';
      } else {
        el.balanceStatusChip.textContent = 'Balanced';
        el.balanceStatusChip.className = 'status-chip chip-balance';
      }

      el.metricTotalCount.textContent = rollup.total_count;
      el.metricDirectCount.textContent = rollup.direct_count;
      el.directCountBadge.textContent = rollup.direct_count;
      el.consolidatedCountBadge.textContent = rollup.total_count;
    }
  }

  function getDepthLabel(depth) {
    switch (depth) {
      case 1: return 'Department';
      case 2: return 'Group';
      case 3: return 'Subgroup';
      case 4: return 'Sub-subgroup';
      case 5: return 'Leaf Subgroup';
      default: return `Level ${depth}`;
    }
  }

  // Load Entries from API
  async function loadEntries() {
    if (!state.activeNodeId) return;

    const includeDesc = state.entriesViewMode === 'consolidated';
    el.thOriginNode.style.display = includeDesc ? 'table-cell' : 'none';

    try {
      const res = await apiFetch(`/api/nodes/${state.activeNodeId}/entries?include_descendants=${includeDesc}&limit=1000`);
      const entries = Array.isArray(res) ? res : (res && res.items ? res.items : []);
      state.rawEntries = entries || [];
      state.selectedEntryIds.clear();
      updateBulkActionBar();

      applyTableFiltersAndSort();
      renderVisualTrendChart(state.rawEntries);
    } catch (err) {
      console.error('Failed to load entries', err);
    }
  }

  // Visual Cash Flow Trend Bar Graph (SVG)
  function renderVisualTrendChart(entries) {
    if (!el.trendsCard) return;
    if (!entries || entries.length === 0) {
      el.trendsCard.classList.add('hidden');
      return;
    }
    el.trendsCard.classList.remove('hidden');

    const monthMap = {};
    for (const e of entries) {
      const m = e.entry_date.substring(0, 7); // "YYYY-MM"
      if (!monthMap[m]) monthMap[m] = { month: m, jama: 0, udhar: 0 };
      if (e.type === 'CREDIT') monthMap[m].jama += e.amount;
      else monthMap[m].udhar += e.amount;
    }

    const periods = Object.values(monthMap).sort((a, b) => a.month.localeCompare(b.month));
    let maxVal = 1000;
    for (const p of periods) {
      maxVal = Math.max(maxVal, p.jama, p.udhar);
    }

    const chartHeight = 85;
    const barWidth = 20;
    const gap = 16;
    const groupWidth = barWidth * 2 + 4;
    const totalWidth = Math.max(periods.length * (groupWidth + gap) + 60, 480);

    const svgBars = periods.map((p, idx) => {
      const x = 30 + idx * (groupWidth + gap);
      const jamaH = p.jama > 0 ? Math.max((p.jama / maxVal) * (chartHeight - 26), 3) : 0;
      const udharH = p.udhar > 0 ? Math.max((p.udhar / maxVal) * (chartHeight - 26), 3) : 0;
      const jamaY = chartHeight - 20 - jamaH;
      const udharY = chartHeight - 20 - udharH;

      const parts = p.month.split('-');
      const label = `${parts[1]}/${parts[0].substring(2)}`;

      return `
        <g class="chart-group">
          <!-- Jama Bar (Green) -->
          ${jamaH > 0 ? `<rect x="${x}" y="${jamaY}" width="${barWidth}" height="${jamaH}" fill="var(--jama)" rx="3"><title>Jama: ${formatMoney(p.jama)} (${p.month})</title></rect>` : ''}
          <!-- Udhar Bar (Red) -->
          ${udharH > 0 ? `<rect x="${x + barWidth + 3}" y="${udharY}" width="${barWidth}" height="${udharH}" fill="var(--udhar)" rx="3"><title>Udhar: ${formatMoney(p.udhar)} (${p.month})</title></rect>` : ''}
          <!-- Month Text -->
          <text x="${x + barWidth}" y="${chartHeight - 4}" text-anchor="middle" font-size="10" fill="var(--text-muted)" font-family="JetBrains Mono">
            ${label}
          </text>
        </g>
      `;
    }).join('');

    el.trendChartContainer.innerHTML = `
      <svg width="100%" height="100%" viewBox="0 0 ${totalWidth} ${chartHeight}" preserveAspectRatio="xMinYMid meet">
        <line x1="10" y1="${chartHeight - 20}" x2="${totalWidth - 10}" y2="${chartHeight - 20}" stroke="var(--border-color)" stroke-width="1" />
        ${svgBars}
      </svg>
    `;
  }

  // Table Filtering, Sorting & Pagination
  function applyTableFiltersAndSort() {
    let result = [...state.rawEntries];

    // 1. Text Search Filter
    const q = state.searchQuery.trim().toLowerCase();
    if (q) {
      result = result.filter(e => {
        const desc = (e.description || '').toLowerCase();
        const ref = (e.reference || '').toLowerCase();
        const node = (e.node_name || '').toLowerCase();
        return desc.includes(q) || ref.includes(q) || node.includes(q);
      });
    }

    // 2. Date Range Filter (Modern Pills)
    const today = new Date();
    if (state.dateFilter === 'month') {
      const curMonth = today.toISOString().substring(0, 7); // YYYY-MM
      result = result.filter(e => e.entry_date.startsWith(curMonth));
    } else if (state.dateFilter === 'quarter') {
      const curQuarter = Math.floor(today.getMonth() / 3);
      const curYear = today.getFullYear();
      result = result.filter(e => {
        const d = new Date(e.entry_date);
        return d.getFullYear() === curYear && Math.floor(d.getMonth() / 3) === curQuarter;
      });
    } else if (state.dateFilter === 'year') {
      const curYear = String(today.getFullYear());
      result = result.filter(e => e.entry_date.startsWith(curYear));
    } else if (state.dateFilter === 'custom') {
      if (state.dateStart) {
        result = result.filter(e => e.entry_date >= state.dateStart);
      }
      if (state.dateEnd) {
        result = result.filter(e => e.entry_date <= state.dateEnd);
      }
    }

    // 3. Type Filter
    if (state.typeFilter !== 'all') {
      result = result.filter(e => e.type === state.typeFilter);
    }

    // 4. Sorting
    result.sort((a, b) => {
      let vA, vB;
      switch (state.sortColumn) {
        case 'entry_date':
          vA = a.entry_date; vB = b.entry_date; break;
        case 'reference':
          vA = (a.reference || '').toLowerCase(); vB = (b.reference || '').toLowerCase(); break;
        case 'description':
          vA = (a.description || '').toLowerCase(); vB = (b.description || '').toLowerCase(); break;
        case 'node_name':
          vA = (a.node_name || '').toLowerCase(); vB = (b.node_name || '').toLowerCase(); break;
        case 'debit':
          vA = a.type === 'DEBIT' ? a.amount : 0; vB = b.type === 'DEBIT' ? b.amount : 0; break;
        case 'credit':
          vA = a.type === 'CREDIT' ? a.amount : 0; vB = b.type === 'CREDIT' ? b.amount : 0; break;
        default:
          vA = a.entry_date; vB = b.entry_date;
      }

      if (vA < vB) return state.sortDirection === 'asc' ? -1 : 1;
      if (vA > vB) return state.sortDirection === 'asc' ? 1 : -1;
      return 0;
    });

    state.filteredEntries = result;
    updateSortIndicators();
    renderCurrentPage();
  }

  function updateSortIndicators() {
    ['entry_date', 'reference', 'description', 'node_name', 'debit', 'credit'].forEach(col => {
      const ind = document.getElementById(`sort-${col}`);
      if (ind) {
        if (state.sortColumn === col) {
          ind.textContent = state.sortDirection === 'asc' ? '▲' : '▼';
        } else {
          ind.textContent = '';
        }
      }
    });
  }

  // Render Table Rows for Current Page
  function renderCurrentPage() {
    const total = state.filteredEntries.length;
    const totalPages = Math.max(1, Math.ceil(total / state.pageSize));

    if (state.currentPage > totalPages) state.currentPage = totalPages;
    if (state.currentPage < 1) state.currentPage = 1;

    const startIdx = (state.currentPage - 1) * state.pageSize;
    const endIdx = Math.min(startIdx + state.pageSize, total);
    const pageRows = state.filteredEntries.slice(startIdx, endIdx);

    // Update Pagination Stats
    el.paginationInfo.textContent = total > 0 ? `Showing ${startIdx + 1}–${endIdx} of ${total}` : 'Showing 0 of 0';
    el.paginationCurrentPage.textContent = `Page ${state.currentPage} of ${totalPages}`;

    el.btnPageFirst.disabled = state.currentPage <= 1;
    el.btnPagePrev.disabled = state.currentPage <= 1;
    el.btnPageNext.disabled = state.currentPage >= totalPages;
    el.btnPageLast.disabled = state.currentPage >= totalPages;

    // Check All Checkbox status
    const allOnPageSelected = pageRows.length > 0 && pageRows.every(r => state.selectedEntryIds.has(r.id));
    el.selectAllCheckbox.checked = allOnPageSelected;

    if (pageRows.length === 0) {
      el.entriesTableBody.innerHTML = `
        <tr>
          <td colspan="8" class="empty-state">
            No transactions found matching your filters.
          </td>
        </tr>
      `;
      return;
    }

    const isConsolidated = state.entriesViewMode === 'consolidated';

    el.entriesTableBody.innerHTML = pageRows.map(e => {
      const isDebit = e.type === 'DEBIT';
      const isSelected = state.selectedEntryIds.has(e.id);
      const debitCol = isDebit ? `<span class="td-amount-num text-udhar">-${formatMoney(e.amount)}</span>` : '<span style="color:var(--text-light)">—</span>';
      const creditCol = !isDebit ? `<span class="td-amount-num text-jama">+${formatMoney(e.amount)}</span>` : '<span style="color:var(--text-light)">—</span>';
      const refPill = e.reference ? `<span class="td-ref-pill">${escapeHTML(e.reference)}</span>` : '<span style="color:var(--text-light)">—</span>';
      const nodeCol = isConsolidated ? `<td><span class="td-node-pill">${escapeHTML(e.node_name || 'Current Node')}</span></td>` : '';

      return `
        <tr class="${isSelected ? 'selected-row' : ''}">
          <td class="col-check">
            <input type="checkbox" class="row-checkbox" data-entry-id="${e.id}" ${isSelected ? 'checked' : ''}>
          </td>
          <td class="col-date td-date-format">${e.entry_date}</td>
          <td class="col-ref">${refPill}</td>
          <td class="col-desc"><strong>${escapeHTML(e.description)}</strong></td>
          ${nodeCol}
          <td class="col-amount text-right">${debitCol}</td>
          <td class="col-amount text-right">${creditCol}</td>
          <td class="col-action text-center no-print">
            <button class="btn-icon-trash" data-del-entry="${e.id}" title="Delete transaction">🗑️</button>
          </td>
        </tr>
      `;
    }).join('');

    // Row Checkbox Listeners
    el.entriesTableBody.querySelectorAll('.row-checkbox').forEach(chk => {
      chk.onchange = () => {
        if (chk.checked) {
          state.selectedEntryIds.add(chk.dataset.entryId);
        } else {
          state.selectedEntryIds.delete(chk.dataset.entryId);
        }
        updateBulkActionBar();
        renderCurrentPage();
      };
    });

    // Delete Buttons
    el.entriesTableBody.querySelectorAll('[data-del-entry]').forEach(btn => {
      btn.onclick = async () => {
        if (!confirm('Are you sure you want to delete this ledger transaction?')) return;
        try {
          await apiFetch(`/api/entries/${btn.dataset.delEntry}`, { method: 'DELETE' });
          showToast('Transaction deleted successfully', 'success');
          await refreshAll();
        } catch (err) {}
      };
    });
  }

  // Bulk Action Bar updates
  function updateBulkActionBar() {
    const count = state.selectedEntryIds.size;
    if (count > 0) {
      el.bulkActionBar.classList.remove('hidden');
      el.bulkSelectedCount.textContent = count;
    } else {
      el.bulkActionBar.classList.add('hidden');
    }
  }

  // Export to CSV Functionality
  function exportEntriesToCSV(entriesToExport, customFilename) {
    if (!entriesToExport || entriesToExport.length === 0) {
      showToast('No entries available to export', 'error');
      return;
    }

    const headers = ['Date', 'Voucher / Ref', 'Description', 'Origin Node', 'Type', 'Debit (Udhar INR)', 'Credit (Jama INR)'];
    const rows = entriesToExport.map(e => {
      const debit = e.type === 'DEBIT' ? e.amount.toFixed(2) : '0.00';
      const credit = e.type === 'CREDIT' ? e.amount.toFixed(2) : '0.00';
      return [
        `"${e.entry_date}"`,
        `"${(e.reference || '').replace(/"/g, '""')}"`,
        `"${(e.description || '').replace(/"/g, '""')}"`,
        `"${(e.node_name || (state.activeNodeDetails ? state.activeNodeDetails.node.name : '')).replace(/"/g, '""')}"`,
        `"${e.type}"`,
        `"${debit}"`,
        `"${credit}"`,
      ].join(',');
    });

    const csvContent = '\uFEFF' + headers.join(',') + '\n' + rows.join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.setAttribute('href', url);
    const nodeName = state.activeNodeDetails ? state.activeNodeDetails.node.name.replace(/[^a-zA-Z0-9]/g, '_') : 'Ledger';
    link.setAttribute('download', customFilename || `Khata_${nodeName}_${new Date().toISOString().split('T')[0]}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    showToast(`Exported ${entriesToExport.length} transactions to CSV!`, 'success');
  }

  // Generate Official Financial Ledger Statement Form
  async function generateOfficialReport(scopeOverride) {
    if (!state.activeNodeId) return;

    const scope = scopeOverride || (el.scopeDirect && el.scopeDirect.checked ? 'direct' : 'consolidated');
    const includeDesc = scope === 'consolidated';

    if (el.modalReportPreviewTarget) {
      el.modalReportPreviewTarget.innerHTML = '<div class="loading-state">Compiling official statement and all entries from ledger...</div>';
    }

    try {
      // 1. Fetch ALL entries for this node (up to 10,000 for complete audit reporting)
      const res = await apiFetch(`/api/nodes/${state.activeNodeId}/entries?include_descendants=${includeDesc}&limit=10000`);
      const allEntries = Array.isArray(res) ? res : (res && res.items ? res.items : []);

      // 2. Fetch node details and ancestors
      let nodeDetails = state.activeNodeDetails;
      if (!nodeDetails || nodeDetails.node.id !== state.activeNodeId) {
        nodeDetails = await apiFetch(`/api/nodes/${state.activeNodeId}`);
      }
      const node = nodeDetails.node;
      const ancestors = nodeDetails.ancestors || [];
      const rollup = nodeDetails.rollup || {};

      // 3. Department name & Hierarchy trail
      const deptName = (ancestors.length > 0 && ancestors[0].name) ? ancestors[0].name : (node.name || 'Main Department');
      const breadcrumbTrail = ancestors.map(a => `${escapeHTML(a.name)} (L${a.depth})`).join(' &gt; ');

      // 4. Sort entries chronologically for formal audit ledger (Date ASC, Created ASC)
      const sortedEntries = [...allEntries].sort((a, b) => {
        if (a.entry_date !== b.entry_date) return a.entry_date.localeCompare(b.entry_date);
        return (a.created_at || '').localeCompare(b.created_at || '');
      });

      // 5. Calculate cumulative Running Balance and Totals
      let runningBalance = 0;
      let totalJama = 0;
      let totalUdhar = 0;

      const rowsHTML = sortedEntries.map((e, index) => {
        const amt = e.amount || (e.amount_paise ? e.amount_paise / 100 : 0);
        let debitStr = '-';
        let creditStr = '-';

        if (e.type === 'CREDIT') {
          totalJama += amt;
          runningBalance += amt;
          creditStr = formatMoney(amt);
        } else {
          totalUdhar += amt;
          runningBalance -= amt;
          debitStr = formatMoney(amt);
        }

        const balSign = runningBalance > 0 ? '+' : '';
        const balColor = runningBalance > 0 ? '#15803d' : (runningBalance < 0 ? '#b91c1c' : '#334155');

        return `
          <tr>
            <td style="text-align: center; color: #64748b; font-weight: 600;">${index + 1}</td>
            <td style="white-space: nowrap; font-weight: 500;">${escapeHTML(e.entry_date)}</td>
            <td style="font-family: monospace; font-size: 0.76rem; color: #475569;">${escapeHTML(e.reference || '—')}</td>
            <td><strong>${escapeHTML(e.description || '—')}</strong></td>
            <td style="font-size: 0.75rem; color: #64748b;">${escapeHTML(e.node_name || node.name)}</td>
            <td style="text-align: right; color: ${debitStr !== '-' ? '#b91c1c' : '#94a3b8'}; font-weight: 600; font-family: monospace;">${debitStr}</td>
            <td style="text-align: right; color: ${creditStr !== '-' ? '#15803d' : '#94a3b8'}; font-weight: 600; font-family: monospace;">${creditStr}</td>
            <td style="text-align: right; color: ${balColor}; font-weight: 700; font-family: monospace;">${balSign}${formatMoney(runningBalance)}</td>
          </tr>
        `;
      }).join('');

      const netBalance = totalJama - totalUdhar;
      const netSign = netBalance > 0 ? '+' : '';
      const netStatus = netBalance > 0 ? 'NET SURPLUS (CREDIT BALANCE)' : (netBalance < 0 ? 'NET DEFICIT (DEBIT BALANCE)' : 'BALANCED');

      const currentUser = state.users.find(u => u.id === state.currentUserId);
      const currentUserName = currentUser ? `${currentUser.name} (${currentUser.role})` : 'Authorized Administrator';
      const nowStr = new Date().toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' });
      const docRef = `EKB-LGR-${node.id.toUpperCase()}-${Math.floor(1000 + Math.random() * 9000)}`;

      // 6. Build Document HTML
      const docHTML = `
        <div class="official-report-document">
          <!-- Institutional Statement Header -->
          <div class="report-statement-header">
            <div class="statement-crest-box">⚖️</div>
            <div class="statement-header-details">
              <h1 class="statement-main-title">ENTERPRISE FINANCIAL LEDGER</h1>
              <div class="statement-meta-line">CONFIDENTIAL & PROPRIETARY • HIERARCHICAL AUDIT STATEMENT</div>
              <div class="statement-tagline">CONSOLIDATED MULTI-DEPARTMENT FINANCIAL ACCOUNTING SYSTEM</div>
              <div class="statement-title-badge">OFFICIAL KHATA LEDGER STATEMENT & AUDIT FORM</div>
            </div>
          </div>

          <!-- Metadata Grid -->
          <div class="report-meta-grid">
            <div class="report-meta-col">
              <div class="meta-row"><span class="meta-lbl">Department:</span> <strong>${escapeHTML(deptName)}</strong></div>
              <div class="meta-row"><span class="meta-lbl">Ledger Head / Account:</span> <strong>${escapeHTML(node.name)}</strong> <span class="rep-depth-badge">Depth ${node.depth}</span></div>
              <div class="meta-row"><span class="meta-lbl">Hierarchy Trail:</span> <span>${breadcrumbTrail}</span></div>
            </div>
            <div class="report-meta-col">
              <div class="meta-row"><span class="meta-lbl">Report Scope:</span> <strong>${includeDesc ? 'Consolidated Roll-up (All Subgroups)' : 'Direct Account Entries Only'}</strong></div>
              <div class="meta-row"><span class="meta-lbl">Total Entries:</span> <strong>${sortedEntries.length} Transactions</strong></div>
              <div class="meta-row"><span class="meta-lbl">Currency:</span> <strong>Indian Rupees (INR - ₹)</strong></div>
            </div>
            <div class="report-meta-col">
              <div class="meta-row"><span class="meta-lbl">Generated On:</span> <strong>${nowStr}</strong></div>
              <div class="meta-row"><span class="meta-lbl">Generated By:</span> <strong>${escapeHTML(currentUserName)}</strong></div>
              <div class="meta-row"><span class="meta-lbl">Document Ref:</span> <span style="font-family: monospace; font-weight: 700;">${docRef}</span></div>
            </div>
          </div>

          <!-- Executive Financial Summary -->
          <div class="report-financial-summary">
            <div class="summary-stat-box jama-border">
              <div class="stat-caption">TOTAL JAMA (CREDITS / INFLOW)</div>
              <div class="stat-val text-jama">${formatMoney(totalJama)}</div>
              <div class="stat-sub">Revenue, Receipts & Inflow</div>
            </div>
            <div class="summary-stat-box udhar-border">
              <div class="stat-caption">TOTAL UDHAR (DEBITS / OUTFLOW)</div>
              <div class="stat-val text-udhar">${formatMoney(totalUdhar)}</div>
              <div class="stat-sub">Expenditures, Bills & Disbursals</div>
            </div>
            <div class="summary-stat-box balance-border">
              <div class="stat-caption">NET CLOSING BALANCE</div>
              <div class="stat-val" style="color: ${netBalance >= 0 ? '#15803d' : '#b91c1c'};">${netSign}${formatMoney(netBalance)}</div>
              <div class="stat-sub" style="font-weight: 700; color: ${netBalance >= 0 ? '#15803d' : '#b91c1c'};">${netStatus}</div>
            </div>
          </div>

          <!-- Detailed Entries Table -->
          <table class="report-table">
            <thead>
              <tr>
                <th style="width: 35px; text-align: center;">#</th>
                <th style="width: 90px;">Date</th>
                <th style="width: 110px;">Voucher / Ref</th>
                <th>Particulars / Description</th>
                <th style="width: 140px;">Ledger Subgroup</th>
                <th style="width: 115px; text-align: right;">Udhar (Debit ₹)</th>
                <th style="width: 115px; text-align: right;">Jama (Credit ₹)</th>
                <th style="width: 125px; text-align: right;">Running Balance (₹)</th>
              </tr>
            </thead>
            <tbody>
              ${sortedEntries.length > 0 ? rowsHTML : '<tr><td colspan="8" style="text-align: center; padding: 20px; color: #64748b;">No transactions recorded in this statement period.</td></tr>'}
            </tbody>
            <tfoot>
              <tr class="report-total-row">
                <td colspan="5" style="text-align: right; font-weight: 800;">GRAND TOTALS / CLOSING BALANCE:</td>
                <td style="text-align: right; font-weight: 800; color: #b91c1c; font-family: monospace;">${formatMoney(totalUdhar)}</td>
                <td style="text-align: right; font-weight: 800; color: #15803d; font-family: monospace;">${formatMoney(totalJama)}</td>
                <td style="text-align: right; font-weight: 900; color: ${netBalance >= 0 ? '#15803d' : '#b91c1c'}; font-family: monospace;">${netSign}${formatMoney(netBalance)}</td>
              </tr>
            </tfoot>
          </table>

          <!-- Official Authorization & Verification Signatures -->
          <div class="report-signatures-grid">
            <div class="sig-block">
              <div class="sig-line"></div>
              <div class="sig-role">Prepared By</div>
              <div class="sig-title">Accounts Officer / Cashier</div>
            </div>
            <div class="sig-block">
              <div class="sig-line"></div>
              <div class="sig-role">Verified & Audited By</div>
              <div class="sig-title">Senior Auditor / Comptroller</div>
            </div>
            <div class="sig-block">
              <div class="sig-line"></div>
              <div class="sig-role">Approved By</div>
              <div class="sig-title">Department Head / Director</div>
              <div class="sig-school">EXECUTIVE FINANCE BOARD</div>
            </div>
            <div class="seal-placeholder">
              OFFICIAL AUDIT SEAL & STAMP
            </div>
          </div>

          <div class="report-footer-disclaimer">
            This is an official computer-generated financial audit statement. Strictly Confidential & Proprietary. Any unauthorized copying, distribution, or alteration is strictly prohibited.
          </div>
        </div>
      `;

      if (el.officialReportContainer) {
        el.officialReportContainer.innerHTML = docHTML;
      }
      if (el.modalReportPreviewTarget) {
        el.modalReportPreviewTarget.innerHTML = docHTML;
      }
    } catch (err) {
      console.error('Failed to generate report', err);
      showToast('Failed to generate report document: ' + err.message, 'error');
    }
  }

  // Refresh tree and current node details
  async function refreshAll() {
    await loadDepartmentTree(state.activeDeptId);
  }

  // Modal Dialog Helpers
  function openModal(modalEl) { modalEl.classList.remove('hidden'); }
  function closeModal(modalEl) { modalEl.classList.add('hidden'); }

  function setupModalListeners() {
    document.querySelectorAll('[data-close]').forEach(btn => {
      btn.onclick = () => {
        document.querySelectorAll('.modal-overlay').forEach(m => m.classList.add('hidden'));
      };
    });

    window.onclick = (e) => {
      if (e.target.classList.contains('modal-overlay')) {
        e.target.classList.add('hidden');
      }
    };
  }

  // Setup Event Listeners
  function setupEventListeners() {
    // Bento Sidebar Capsule Navigation
    if (el.sidebarNavOverview) el.sidebarNavOverview.onclick = () => setView('overview');
    if (el.sidebarNavLedger) el.sidebarNavLedger.onclick = () => setView('ledger');
    if (el.sidebarNavReport) el.sidebarNavReport.onclick = () => { el.btnPrintPDF.click(); };
    if (el.sidebarNavUsers) el.sidebarNavUsers.onclick = () => { el.userSelect.focus(); };
    if (el.sidebarHelpBtn) el.sidebarHelpBtn.onclick = () => {
      showToast('Enterprise Khata Book • Up to 5 Hierarchy Levels • Materialized Path • Serialized DB', 'info');
    };

    // Bento Quick Action Buttons
    if (el.btnSwitchToFullLedger) el.btnSwitchToFullLedger.onclick = () => setView('ledger');
    if (el.btnBentoOpenReport) el.btnBentoOpenReport.onclick = () => { el.btnPrintPDF.click(); };

    // Synchronize Top Search Pill with Table Search
    if (el.topNavSearchInput) {
      el.topNavSearchInput.oninput = (e) => {
        const val = e.target.value;
        if (el.tableSearchInput) el.tableSearchInput.value = val;
        state.searchQuery = val.trim();
        state.currentPage = 1;
        applyTableFiltersAndSort();
      };
    }

    // Navigation Tabs
    if (el.navViewOverview) el.navViewOverview.onclick = () => setView('overview');
    if (el.navViewLedger) el.navViewLedger.onclick = () => setView('ledger');

    // User Switcher
    el.userSelect.onchange = async () => {
      state.currentUserId = el.userSelect.value;
      localStorage.setItem('khata_user_id', state.currentUserId);
      updateUserBadge();
      showToast(`Switched user context to: ${el.userSelect.options[el.userSelect.selectedIndex].text}`, 'info');
      state.activeDeptId = null;
      state.activeNodeId = null;
      if (state.currentView === 'overview') {
        await loadGlobalOverview();
      } else {
        await loadDepartments();
      }
    };

    // Department Dropdown Switcher
    el.deptSelect.onchange = async () => {
      state.activeDeptId = el.deptSelect.value;
      state.activeNodeId = null;
      await loadDepartmentTree(state.activeDeptId);
    };

    // Table Search
    el.tableSearchInput.oninput = () => {
      state.searchQuery = el.tableSearchInput.value;
      el.btnClearSearch.classList.toggle('hidden', !state.searchQuery);
      state.currentPage = 1;
      applyTableFiltersAndSort();
    };

    el.btnClearSearch.onclick = () => {
      el.tableSearchInput.value = '';
      state.searchQuery = '';
      el.btnClearSearch.classList.add('hidden');
      state.currentPage = 1;
      applyTableFiltersAndSort();
    };

    // Modern Date Filter Pills
    document.querySelectorAll('[data-date-filter]').forEach(btn => {
      btn.onclick = () => {
        document.querySelectorAll('[data-date-filter]').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');

        state.dateFilter = btn.dataset.dateFilter;
        el.customDateRangeBox.classList.toggle('hidden', state.dateFilter !== 'custom');
        state.currentPage = 1;
        applyTableFiltersAndSort();
      };
    });

    el.dateFilterStart.onchange = () => {
      state.dateStart = el.dateFilterStart.value;
      state.currentPage = 1;
      applyTableFiltersAndSort();
    };

    el.dateFilterEnd.onchange = () => {
      state.dateEnd = el.dateFilterEnd.value;
      state.currentPage = 1;
      applyTableFiltersAndSort();
    };

    // Modern Type Filter Pills
    document.querySelectorAll('[data-type-filter]').forEach(btn => {
      btn.onclick = () => {
        document.querySelectorAll('[data-type-filter]').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');

        state.typeFilter = btn.dataset.typeFilter;
        state.currentPage = 1;
        applyTableFiltersAndSort();
      };
    });

    // Sorting Headers
    document.querySelectorAll('.khata-data-table th.sortable').forEach(th => {
      th.onclick = () => {
        const col = th.dataset.sort;
        if (state.sortColumn === col) {
          state.sortDirection = state.sortDirection === 'asc' ? 'desc' : 'asc';
        } else {
          state.sortColumn = col;
          state.sortDirection = 'desc';
        }
        applyTableFiltersAndSort();
      };
    });

    // Select All Checkbox
    el.selectAllCheckbox.onchange = () => {
      const isChecked = el.selectAllCheckbox.checked;
      const startIdx = (state.currentPage - 1) * state.pageSize;
      const endIdx = Math.min(startIdx + state.pageSize, state.filteredEntries.length);
      const pageRows = state.filteredEntries.slice(startIdx, endIdx);

      pageRows.forEach(r => {
        if (isChecked) state.selectedEntryIds.add(r.id);
        else state.selectedEntryIds.delete(r.id);
      });

      updateBulkActionBar();
      renderCurrentPage();
    };

    // Pagination Controls
    el.pageSizeSelect.onchange = () => {
      state.pageSize = parseInt(el.pageSizeSelect.value, 10);
      state.currentPage = 1;
      renderCurrentPage();
    };

    el.btnPageFirst.onclick = () => { state.currentPage = 1; renderCurrentPage(); };
    el.btnPagePrev.onclick = () => { if (state.currentPage > 1) { state.currentPage--; renderCurrentPage(); } };
    el.btnPageNext.onclick = () => { state.currentPage++; renderCurrentPage(); };
    el.btnPageLast.onclick = () => {
      state.currentPage = Math.ceil(state.filteredEntries.length / state.pageSize);
      renderCurrentPage();
    };

    // Export CSV via Go streaming endpoint
    el.btnExportCSV.onclick = () => {
      if (!state.activeNodeId) return;
      const includeDesc = state.entriesViewMode === 'consolidated';
      const searchParam = state.searchQuery ? `&q=${encodeURIComponent(state.searchQuery)}` : '';
      const periodParam = state.dateFilter !== 'all' ? `&period=${encodeURIComponent(state.dateFilter)}` : '';
      const fromParam = state.dateStart ? `&from_date=${encodeURIComponent(state.dateStart)}` : '';
      const toParam = state.dateEnd ? `&to_date=${encodeURIComponent(state.dateEnd)}` : '';
      const typeParam = state.typeFilter !== 'all' ? `&type=${encodeURIComponent(state.typeFilter)}` : '';
      const url = `/api/nodes/${state.activeNodeId}/export?include_descendants=${includeDesc}${searchParam}${periodParam}${fromParam}${toParam}${typeParam}`;
      window.location.href = url;
    };

    // Generate and Preview Official Financial Ledger Statement Form
    el.btnPrintPDF.onclick = async () => {
      if (!state.activeNodeId) {
        showToast('Please select a department or group node first', 'info');
        return;
      }
      if (state.entriesViewMode === 'consolidated') {
        if (el.scopeConsolidated) el.scopeConsolidated.checked = true;
      } else {
        if (el.scopeDirect) el.scopeDirect.checked = true;
      }
      openModal(el.reportModal);
      await generateOfficialReport();
    };

    // Print triggers from inside preview modal
    if (el.btnTriggerPrint) {
      el.btnTriggerPrint.onclick = () => {
        window.print();
      };
    }
    if (el.btnModalPrintDoc) {
      el.btnModalPrintDoc.onclick = () => {
        window.print();
      };
    }

    // Scope Radio changes inside modal
    if (el.scopeConsolidated) {
      el.scopeConsolidated.onchange = () => generateOfficialReport('consolidated');
    }
    if (el.scopeDirect) {
      el.scopeDirect.onchange = () => generateOfficialReport('direct');
    }

    // Also handle native Ctrl+P / beforeprint event
    window.addEventListener('beforeprint', () => {
      generateOfficialReport();
    });

    // Bulk Action Bar: Clear Selection
    el.btnCancelBulk.onclick = () => {
      state.selectedEntryIds.clear();
      updateBulkActionBar();
      renderCurrentPage();
    };

    // Bulk Action Bar: Export Selected
    el.btnBulkExport.onclick = () => {
      const selected = state.rawEntries.filter(e => state.selectedEntryIds.has(e.id));
      exportEntriesToCSV(selected, `Khata_Selected_${state.selectedEntryIds.size}_Entries.csv`);
    };

    // Bulk Action Bar: Bulk Delete
    el.btnBulkDelete.onclick = async () => {
      const count = state.selectedEntryIds.size;
      if (!confirm(`Are you sure you want to permanently delete ${count} selected transactions?`)) return;

      try {
        await apiFetch('/api/entries/bulk-delete', {
          method: 'POST',
          body: JSON.stringify({ entry_ids: Array.from(state.selectedEntryIds) }),
        });
        showToast(`Successfully deleted ${count} transactions`, 'success');
        state.selectedEntryIds.clear();
        await refreshAll();
      } catch (err) {}
    };

    // Bulk Action Bar: Bulk Move
    el.btnBulkMove.onclick = () => {
      const count = state.selectedEntryIds.size;
      el.bulkMoveCountLabel.textContent = count;

      const flat = collectFlatNodes(state.activeTree);
      el.bulkMoveTargetSelect.innerHTML = flat
        .map(n => `<option value="${n.id}">${'—'.repeat(n.depth - 1)} ${escapeHTML(n.name)} (Depth ${n.depth})</option>`)
        .join('');

      openModal(el.bulkMoveModal);
    };

    el.bulkMoveForm.onsubmit = async (e) => {
      e.preventDefault();
      const targetId = el.bulkMoveTargetSelect.value;
      const count = state.selectedEntryIds.size;

      try {
        await apiFetch('/api/entries/bulk-move', {
          method: 'POST',
          body: JSON.stringify({
            entry_ids: Array.from(state.selectedEntryIds),
            target_node_id: targetId,
          }),
        });
        closeModal(el.bulkMoveModal);
        showToast(`Successfully moved ${count} transactions!`, 'success');
        state.selectedEntryIds.clear();
        await refreshAll();
      } catch (err) {}
    };

    // Create New Department
    const openNewDeptModal = () => {
      el.modalNodeTitle.textContent = 'Create New Department (Level 1 Root)';
      el.modalParentName.textContent = 'None (Root Level)';
      el.modalNewDepthBadge.textContent = 'Depth 1 (Department)';
      el.modalNewDepthBadge.className = 'level-indicator level-1';
      el.nodeNameInput.value = '';
      el.createNodeForm.dataset.parentId = '';
      openModal(el.createNodeModal);
    };

    el.btnNewDept.onclick = openNewDeptModal;
    if (el.btnOverviewNewDept) el.btnOverviewNewDept.onclick = openNewDeptModal;

    // Add Subgroup under Current Node
    el.btnAddSubgroup.onclick = () => {
      if (!state.activeNodeDetails) return;
      const parent = state.activeNodeDetails.node;
      if (parent.depth >= MAX_DEPTH) {
        showToast('Maximum depth limit of 5 is reached!', 'error');
        return;
      }

      const nextDepth = parent.depth + 1;
      el.modalNodeTitle.textContent = `Add Child Node under "${parent.name}"`;
      el.modalParentName.textContent = `${parent.name} (Depth ${parent.depth})`;
      el.modalNewDepthBadge.textContent = `Depth ${nextDepth} (${getDepthLabel(nextDepth)})`;
      el.modalNewDepthBadge.className = `level-indicator level-${nextDepth}`;
      el.nodeNameInput.value = '';
      el.createNodeForm.dataset.parentId = parent.id;
      openModal(el.createNodeModal);
    };

    // Submit Create Node Form
    el.createNodeForm.onsubmit = async (e) => {
      e.preventDefault();
      const name = el.nodeNameInput.value.trim();
      const parentId = el.createNodeForm.dataset.parentId || null;

      try {
        const payload = { name, parent_id: parentId ? parentId : null };
        const newNode = await apiFetch('/api/nodes', {
          method: 'POST',
          body: JSON.stringify(payload),
        });

        closeModal(el.createNodeModal);
        showToast(`Created node "${newNode.name}" at Depth ${newNode.depth}!`, 'success');

        if (!parentId) {
          state.activeDeptId = newNode.id;
          await loadDepartments();
          if (state.currentView === 'overview') await loadGlobalOverview();
        } else {
          state.activeNodeId = newNode.id;
          await refreshAll();
        }
      } catch (err) {}
    };

    // Rename Node
    el.btnRenameNode.onclick = () => {
      if (!state.activeNodeDetails) return;
      el.renameNodeInput.value = state.activeNodeDetails.node.name;
      openModal(el.renameNodeModal);
    };

    el.renameNodeForm.onsubmit = async (e) => {
      e.preventDefault();
      const name = el.renameNodeInput.value.trim();
      try {
        await apiFetch(`/api/nodes/${state.activeNodeId}/rename`, {
          method: 'PUT',
          body: JSON.stringify({ name }),
        });
        closeModal(el.renameNodeModal);
        showToast('Node renamed successfully', 'success');
        await refreshAll();
      } catch (err) {}
    };

    // Move Node
    el.btnMoveNode.onclick = () => {
      if (!state.activeNodeDetails) return;
      const currentNode = state.activeNodeDetails.node;
      el.moveNodeTargetName.textContent = `"${currentNode.name}" (Depth ${currentNode.depth})`;

      const activeTreeNode = findNodeInTree(state.activeTree, currentNode.id);
      const subtreeMaxDepth = getSubtreeMaxDepth(activeTreeNode);
      const subtreeHeight = subtreeMaxDepth - currentNode.depth;
      el.moveSubtreeHeight.textContent = subtreeHeight;

      const flat = collectFlatNodes(state.activeTree);
      const descendantIds = new Set(collectFlatNodes(activeTreeNode).map(n => n.id));

      const validCandidates = flat.filter(n => {
        if (descendantIds.has(n.id)) return false;
        const resultingMaxDepth = n.depth + 1 + subtreeHeight;
        return resultingMaxDepth <= MAX_DEPTH;
      });

      if (validCandidates.length === 0) {
        showToast('No valid target nodes available that would satisfy MAX_DEPTH (5) limit!', 'error');
        return;
      }

      el.moveNewParentSelect.innerHTML = validCandidates
        .map(n => `<option value="${n.id}" data-depth="${n.depth}">${'—'.repeat(n.depth - 1)} ${n.name} (Depth ${n.depth})</option>`)
        .join('');

      el.moveTargetDepth.textContent = validCandidates[0].depth;
      openModal(el.moveNodeModal);
    };

    el.moveNewParentSelect.onchange = () => {
      const selected = el.moveNewParentSelect.options[el.moveNewParentSelect.selectedIndex];
      el.moveTargetDepth.textContent = selected.dataset.depth;
    };

    el.moveNodeForm.onsubmit = async (e) => {
      e.preventDefault();
      const newParentId = el.moveNewParentSelect.value;
      try {
        await apiFetch(`/api/nodes/${state.activeNodeId}/move`, {
          method: 'POST',
          body: JSON.stringify({ new_parent_id: newParentId }),
        });
        closeModal(el.moveNodeModal);
        showToast('Node and its subtree moved successfully', 'success');
        await refreshAll();
      } catch (err) {}
    };

    // Delete Node
    el.btnDeleteNode.onclick = async () => {
      if (!state.activeNodeDetails) return;
      const node = state.activeNodeDetails.node;
      const isDept = node.depth === 1;
      const confirmMsg = isDept
        ? `Are you sure you want to delete the entire department "${node.name}" and ALL its subgroups and ledger entries?`
        : `Are you sure you want to delete "${node.name}" and all its child subgroups and entries?`;

      if (!confirm(confirmMsg)) return;

      try {
        await apiFetch(`/api/nodes/${node.id}`, { method: 'DELETE' });
        showToast('Node deleted successfully', 'success');
        if (isDept) {
          state.activeDeptId = null;
          state.activeNodeId = null;
          await loadDepartments();
        } else {
          state.activeNodeId = null;
          await refreshAll();
        }
      } catch (err) {}
    };

    // Record Khata Entry
    el.btnNewEntry.onclick = () => {
      el.entryAmount.value = '';
      el.entryDate.value = new Date().toISOString().split('T')[0];
      el.entryReference.value = '';
      el.entryDescription.value = '';
      openModal(el.createEntryModal);
    };

    el.createEntryForm.onsubmit = async (e) => {
      e.preventDefault();
      const type = document.querySelector('input[name="entryType"]:checked').value;
      const amount = parseFloat(el.entryAmount.value);
      const entryDate = el.entryDate.value;
      const reference = el.entryReference.value.trim();
      const description = el.entryDescription.value.trim();

      try {
        await apiFetch(`/api/nodes/${state.activeNodeId}/entries`, {
          method: 'POST',
          body: JSON.stringify({
            type,
            amount,
            entry_date: entryDate,
            reference,
            description,
          }),
        });
        closeModal(el.createEntryModal);
        showToast(`Recorded ${type === 'CREDIT' ? 'Jama (+)' : 'Udhar (-)'} of ${formatMoney(amount)}!`, 'success');
        await refreshAll();
      } catch (err) {}
    };

    // Tab Switchers: Direct vs Consolidated
    el.tabDirect.onclick = () => {
      state.entriesViewMode = 'direct';
      el.tabDirect.classList.add('active');
      el.tabConsolidated.classList.remove('active');
      loadEntries();
    };

    el.tabConsolidated.onclick = () => {
      state.entriesViewMode = 'consolidated';
      el.tabConsolidated.classList.add('active');
      el.tabDirect.classList.remove('active');
      loadEntries();
    };
  }

  function escapeHTML(str) {
    if (!str) return '';
    return str.replace(/[&<>'"]/g, tag => ({
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      "'": '&#39;',
      '"': '&quot;',
    }[tag] || tag));
  }

  // Launch app when DOM is ready
  document.addEventListener('DOMContentLoaded', init);
})();
