// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

// @ts-nocheck
import React, { useState, useEffect, useRef } from 'react';
import './AIChatbot.css';

const navigateToPath = (path) => {
  if (!path) {
    return;
  }
  window.location.href = `${window.location.origin}${path}`;
};

const buildAuthHeaders = () => {
  const token = localStorage.getItem('token');
  const headers = {
    'Content-Type': 'application/json'
  };

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  return headers;
};

const deriveModuleKey = (pathname = '') => {
  if (pathname.startsWith('/deploy')) {
    return 'deploy';
  }
  if (pathname.startsWith('/alarm') || pathname.startsWith('/alert')) {
    return 'alert';
  }
  if (pathname.startsWith('/security')) {
    return 'security';
  }
  if (pathname.startsWith('/cmdb')) {
    return 'cmdb';
  }
  if (pathname.startsWith('/consul')) {
    return 'consul';
  }
  if (pathname.startsWith('/monitor')) {
    return 'monitor';
  }
  if (pathname.startsWith('/system')) {
    return 'system';
  }
  if (pathname.startsWith('/admin')) {
    return 'admin';
  }
  return '';
};

const buildPageContext = () => {
  const pathname = window.location?.pathname || '';
  const title = document?.title || '';
  const selectedRecordId = document?.body?.dataset?.assistantRecordId;
  const objectType = document?.body?.dataset?.assistantObjectType;
  const objectId = document?.body?.dataset?.assistantObjectId;
  const filtersJSON = document?.body?.dataset?.assistantFilters;

  const pageContext = {
    pagePath: pathname,
    moduleKey: deriveModuleKey(pathname),
    pageTitle: title
  };

  if (objectType) {
    pageContext.objectType = objectType;
  }
  if (objectId) {
    pageContext.objectId = objectId;
  }
  if (selectedRecordId) {
    pageContext.selectedRecordIds = [selectedRecordId];
  }
  if (filtersJSON) {
    try {
      const filters = JSON.parse(filtersJSON);
      if (filters && typeof filters === 'object' && Object.keys(filters).length > 0) {
        pageContext.filters = filters;
      }
    } catch (error) {
      console.warn('failed to parse assistant filters', error);
    }
  }

  return pageContext;
};

const parseAssistantSummary = (text = '') => {
  const lines = text.split('\n').map((line) => line.trim()).filter(Boolean);
  const cardLines = lines.filter((line) => line.startsWith('- '));
  if (!cardLines.length) {
    return { summary: text, items: [] };
  }

  const summaryLines = [];
  const items = [];

  lines.forEach((line) => {
    if (line.startsWith('- ')) {
      items.push(line.slice(2));
    } else {
      summaryLines.push(line);
    }
  });

  return {
    summary: summaryLines.join('\n'),
    items
  };
};

const formatIntentLabel = (intent = '') => {
  const labels = {
    page_navigation: '页面导航',
    readonly_query: '只读查询',
    knowledge_qa: '知识问答',
    fallback: '智能问答',
    system_notice: '系统提示'
  };

  return labels[intent] || intent;
};

const shouldShowIntentBadge = (intent = '') => intent === 'system_notice';

const shouldShowCitations = (message) => {
  if (!message || !Array.isArray(message.citations) || !message.citations.length) {
    return false;
  }
  return message.intent === 'knowledge_qa';
};

const formatSessionTime = (value) => {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return date.toLocaleString([], {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
};

const readResponseBody = async (response) => {
  const contentType = response.headers.get('Content-Type') || '';
  if (contentType.includes('application/json')) {
    return response.json();
  }

  const text = await response.text();
  return {
    error: text || `HTTP ${response.status}`
  };
};

const AIChatbot = () => {
  const [isAuthenticated, setIsAuthenticated] = useState(Boolean(localStorage.getItem('token')));
  const [isOpen, setIsOpen] = useState(false);
  const [messages, setMessages] = useState([]);
  const [inputValue, setInputValue] = useState('');
  const [isMinimized, setIsMinimized] = useState(false);
  const [isMaximized, setIsMaximized] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [sessions, setSessions] = useState([]);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [sessionPanelOpen, setSessionPanelOpen] = useState(false);
  const [sessionQuery, setSessionQuery] = useState('');
  const [sessionStatus, setSessionStatus] = useState('active');
  const [sessionPage, setSessionPage] = useState(1);
  const [sessionHasMore, setSessionHasMore] = useState(false);
  const [messagePage, setMessagePage] = useState(1);
  const [messageHasMore, setMessageHasMore] = useState(false);
  const [renamingSessionId, setRenamingSessionId] = useState(null);
  const messagesEndRef = useRef(null);

  const sendPrompt = async (promptText) => {
    const currentInput = String(promptText || '').trim();
    if (!currentInput) return;

    setIsLoading(true);

    let activeSessionId = sessionId;
    if (!activeSessionId) {
      activeSessionId = await ensureSession();
    }
    if (!activeSessionId) {
      setIsLoading(false);
      return;
    }

    const userMessage = {
      id: Date.now(),
      text: currentInput,
      sender: 'user',
      timestamp: new Date().toLocaleTimeString()
    };

    setMessages((prev) => [...prev, userMessage]);
    setInputValue('');

    const aiResponse = await getAIResponse(currentInput, activeSessionId);

    const aiMessage = {
      id: aiResponse.messageId || Date.now() + 1,
      text: aiResponse.answer,
      sender: 'ai',
      timestamp: new Date().toLocaleTimeString(),
      intent: aiResponse.intent || '',
      model: aiResponse.model || '',
      resultCards: aiResponse.resultCards || [],
      citations: aiResponse.citations || [],
      actions: aiResponse.actions || []
    };

    setMessages((prev) => [...prev, aiMessage]);
    setIsLoading(false);
    fetchSessions({ page: 1, query: sessionQuery, status: sessionStatus, append: false });
  };

  const appendSystemMessage = (text) => {
    setMessages((prev) => [
      ...prev,
      {
        id: `local-${Date.now()}`,
        text,
        sender: 'ai',
        timestamp: new Date().toLocaleTimeString(),
        intent: 'system_notice',
        resultCards: [],
        citations: [],
        actions: []
      }
    ]);
  };

  const fetchSessions = async ({ page = 1, query = sessionQuery, status = sessionStatus, append = false } = {}) => {
    const token = localStorage.getItem('token');
    if (!token) {
      setSessions([]);
      setIsAuthenticated(false);
      return [];
    }

    setIsAuthenticated(true);
    setSessionsLoading(true);
    try {
      const params = new URLSearchParams({
        page: String(page),
        limit: '10'
      });
      if (query.trim()) {
        params.set('query', query.trim());
      }
      if (status && status !== 'all') {
        params.set('status', status);
      }

      const response = await fetch(`/api/assistant/sessions?${params.toString()}`, {
        headers: buildAuthHeaders()
      });
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || '加载助手会话失败');
      }
      const items = data.sessions || [];
      setSessions((prev) => append ? [...prev, ...items] : items);
      setSessionPage(data.page || page);
      setSessionHasMore(Boolean(data.hasMore));
      return items;
    } catch (error) {
      console.error('加载助手会话失败:', error);
      return [];
    } finally {
      setSessionsLoading(false);
    }
  };

  const ensureSession = async () => {
    const token = localStorage.getItem('token');
    if (!token) {
      setIsAuthenticated(false);
      appendSystemMessage('当前登录状态无效，请重新登录后再使用运维小助手。');
      return null;
    }

    setIsAuthenticated(true);

    if (sessionId) {
      return sessionId;
    }

    try {
      const sessions = await fetchSessions({ page: 1, query: '', status: 'active', append: false });
      let id = null;
      if (sessions.length) {
        id = sessions[0].sessionId;
      } else {
        id = await createSession();
      }

      if (id) {
        setSessionId(id);
        await loadSessionMessages(id, 1);
      } else {
        appendSystemMessage('助手会话初始化失败，请稍后重试。');
      }

      return id;
    } catch (error) {
      console.error('初始化助手会话失败:', error);
      appendSystemMessage('助手会话初始化失败，请检查登录状态或稍后重试。');
      return null;
    }
  };

  // 创建会话的函数
  const createSession = async (forceNew = false) => {
    try {
      const response = await fetch('/api/assistant/sessions', {
        method: 'POST',
        headers: buildAuthHeaders(),
        body: JSON.stringify({
          scene: 'web',
          userAgent: navigator.userAgent,
          ipAddress: '127.0.0.1',
          forceNew
        })
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || '创建会话失败');
      }
      return data.session?.sessionId || null;
    } catch (error) {
      console.error('创建会话失败:', error);
      return null;
    }
  };

  // 从当前站点后端获取 AI 响应
  const getAIResponse = async (userMessage, sessionId) => {
    setIsLoading(true);

    try {
      const response = await fetch('/api/assistant/messages', {
        method: 'POST',
        headers: buildAuthHeaders(),
        body: JSON.stringify({
          sessionId: sessionId,
          message: userMessage,
          pageContext: buildPageContext()
        })
      });

      const data = await response.json();

      if (response.ok && data.answer) {
        setIsLoading(false);
        return data;
      } else {
        throw new Error(data.error || '未知错误');
      }
    } catch (error) {
      console.error('获取AI响应失败:', error);
      setIsLoading(false);

      // 返回更具体的错误消息
      return {
        answer: "抱歉，AI助手暂时无法响应。请确保AI助手服务已启动。错误: " + error.message,
        citations: [],
        actions: []
      };
    }
  };

  // 在组件中保存sessionId
  const [sessionId, setSessionId] = useState(null);

  const loadSessionMessages = async (targetSessionId, page = 1) => {
    const response = await fetch(`/api/assistant/sessions/${targetSessionId}/messages?page=${page}&limit=20`, {
      headers: buildAuthHeaders()
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || '加载历史消息失败');
    }

    const mappedMessages = (data.messages || []).map((message) => ({
      id: message.messageId,
      text: message.text,
      sender: message.role === 'assistant' ? 'ai' : 'user',
      timestamp: new Date(message.createdAt).toLocaleTimeString(),
      intent: message.intent || '',
      model: message.model || '',
      resultCards: message.resultCards || [],
      citations: message.citations || [],
      actions: message.actions || []
    }));
    setMessages(page > 1 ? (prev) => [...mappedMessages, ...prev] : mappedMessages);
    setSessionId(targetSessionId);
    setMessagePage(data.page || page);
    setMessageHasMore(Boolean(data.hasMore));
  };

  const handleSelectSession = async (targetSessionId) => {
    if (!targetSessionId || targetSessionId === sessionId) {
      setSessionPanelOpen(false);
      return;
    }

    try {
      await loadSessionMessages(targetSessionId, 1);
      setSessionPanelOpen(false);
    } catch (error) {
      console.error('切换助手会话失败:', error);
      appendSystemMessage('切换会话失败，请稍后重试。');
    }
  };

  const handleCreateSession = async () => {
    const newSessionId = await createSession(true);
    if (!newSessionId) {
      appendSystemMessage('新建会话失败，请稍后重试。');
      return;
    }

    setSessionId(newSessionId);
    setMessages([]);
    setSessionPanelOpen(false);
    setSessionQuery('');
    setSessionStatus('active');
    await fetchSessions({ page: 1, query: '', status: 'active', append: false });
  };

  const updateSession = async (targetSessionId, payload) => {
    const response = await fetch(`/api/assistant/sessions/${targetSessionId}`, {
      method: 'PATCH',
      headers: buildAuthHeaders(),
      body: JSON.stringify(payload)
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || '更新会话失败');
    }
    return data.session;
  };

  const handlePinSession = async (session) => {
    try {
      await updateSession(session.sessionId, { pinned: !session.pinned });
      await fetchSessions({ page: 1, query: sessionQuery, status: sessionStatus, append: false });
    } catch (error) {
      console.error('更新置顶状态失败:', error);
      appendSystemMessage('更新置顶状态失败，请稍后重试。');
    }
  };

  const handleRenameSession = async (session) => {
    const nextTitle = window.prompt('输入新的会话标题', session.title || session.summary || '');
    if (nextTitle == null) {
      return;
    }

    setRenamingSessionId(session.sessionId);
    try {
      await updateSession(session.sessionId, { title: nextTitle });
      await fetchSessions({ page: 1, query: sessionQuery, status: sessionStatus, append: false });
    } catch (error) {
      console.error('重命名会话失败:', error);
      appendSystemMessage('重命名会话失败，请稍后重试。');
    } finally {
      setRenamingSessionId(null);
    }
  };

  const handleArchiveSession = async (targetSessionId) => {
    try {
      await updateSession(targetSessionId, { status: 'archived' });
      if (targetSessionId === sessionId) {
        setSessionId(null);
        setMessages([]);
        setMessagePage(1);
        setMessageHasMore(false);
      }
      await fetchSessions({ page: 1, query: sessionQuery, status: sessionStatus, append: false });
    } catch (error) {
      console.error('归档会话失败:', error);
      appendSystemMessage('归档会话失败，请稍后重试。');
    }
  };

  const handleLoadOlderMessages = async () => {
    if (!sessionId || !messageHasMore || isLoading) {
      return;
    }
    try {
      await loadSessionMessages(sessionId, messagePage + 1);
    } catch (error) {
      console.error('加载更早消息失败:', error);
      appendSystemMessage('加载更早消息失败，请稍后重试。');
    }
  };

  // 组件加载时创建会话
  useEffect(() => {
    const syncAuthState = () => {
      setIsAuthenticated(Boolean(localStorage.getItem('token')));
    };

    syncAuthState();
    window.addEventListener('storage', syncAuthState);
    window.addEventListener('focus', syncAuthState);

    return () => {
      window.removeEventListener('storage', syncAuthState);
      window.removeEventListener('focus', syncAuthState);
    };
  }, []);

  useEffect(() => {
    if (isOpen && !sessionId && localStorage.getItem('token')) {
      ensureSession();
    }
  }, [isOpen, sessionId]);

  useEffect(() => {
    if (isOpen && localStorage.getItem('token')) {
      fetchSessions({ page: 1, query: sessionQuery, append: false });
    }
  }, [isOpen]);

  useEffect(() => {
    const handleAssistantPrompt = async (event) => {
      const query = event?.detail?.query;
      if (!query || isLoading) {
        return;
      }

      if (!isOpen) {
        setIsOpen(true);
        setIsMinimized(false);
        setIsMaximized(false);
      } else if (isMinimized) {
        setIsMinimized(false);
      }

      await sendPrompt(query);
    };

    window.addEventListener('ops-assistant:prompt', handleAssistantPrompt);
    return () => window.removeEventListener('ops-assistant:prompt', handleAssistantPrompt);
  }, [isLoading, isMinimized, isOpen, sessionId, sessionQuery, sessionStatus]);

  const sendMessage = async () => {
    if (!inputValue.trim()) return;
    await sendPrompt(inputValue);
  };

  const handleSessionSearch = async (event) => {
    const value = event.target.value;
    setSessionQuery(value);
    await fetchSessions({ page: 1, query: value, status: sessionStatus, append: false });
  };

  const handleLoadMoreSessions = async () => {
    if (!sessionHasMore || sessionsLoading) {
      return;
    }
    await fetchSessions({ page: sessionPage + 1, query: sessionQuery, status: sessionStatus, append: true });
  };

  const handleSessionStatusChange = async (status) => {
    setSessionStatus(status);
    await fetchSessions({ page: 1, query: sessionQuery, status, append: false });
  };

  const handleDeleteSession = async (targetSessionId) => {
    if (!window.confirm('确定删除该会话及其全部消息吗？')) {
      return;
    }

    try {
      const response = await fetch(`/api/assistant/sessions/${targetSessionId}`, {
        method: 'DELETE',
        headers: buildAuthHeaders()
      });
      const data = await readResponseBody(response);
      if (!response.ok) {
        throw new Error(data.error || '删除会话失败');
      }

      setSessions((prev) => prev.filter((session) => session.sessionId !== targetSessionId));
      if (targetSessionId === sessionId) {
        setSessionId(null);
        setMessages([]);
        setMessagePage(1);
        setMessageHasMore(false);
      }

      fetchSessions({ page: 1, query: sessionQuery, status: sessionStatus, append: false }).catch((error) => {
        console.error('删除后刷新会话列表失败:', error);
      });
    } catch (error) {
      console.error('删除会话失败:', error);
      appendSystemMessage(`删除会话失败：${error.message || '请稍后重试。'}`);
    }
  };

  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  // 当聊天窗口打开时，重置未读计数
  useEffect(() => {
    if (isOpen) {
      setUnreadCount(0);
    }
  }, [isOpen]);

  // 滚动到底部
  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const toggleMinimize = () => {
    // 如果当前是最大化状态，则先退出最大化
    if (isMaximized) {
      setIsMaximized(false);
    }
    setIsMinimized(!isMinimized);
    if (!isMinimized && unreadCount > 0) {
      setUnreadCount(0);
    }
  };

  const toggleMaximize = () => {
    // 如果当前是最小化状态，则先恢复
    if (isMinimized) {
      setIsMinimized(false);
    }
    setIsMaximized(!isMaximized);
  };

  const closeChat = () => {
    setIsOpen(false);
    setIsMinimized(false);
    setIsMaximized(false);
    setUnreadCount(0);
  };

  const toggleChat = () => {
    if (!isAuthenticated) {
      return;
    }
    if (!isOpen) {
      setIsOpen(true);
      setIsMinimized(false);
      setIsMaximized(false);
    } else {
      if (isMinimized) {
        setIsMinimized(false);
      } else {
        closeChat();
      }
    }
  };

  if (!isAuthenticated) {
    return null;
  }

  // 函数：将包含 [NAVIGATE:] 标签的文本转换为带链接的元素
  const renderMessageText = (text) => {
    // 正则表达式匹配 [NAVIGATE:path] 格式的导航标签
    const navRegex = /\[NAVIGATE:(\/[^\]]+)\]/g;
    const parts = [];
    let lastIndex = 0;
    let match;

    while ((match = navRegex.exec(text)) !== null) {
      // 添加匹配前的文本
      if (match.index > lastIndex) {
        parts.push(text.substring(lastIndex, match.index));
      }

      // 添加可点击的导航链接
      const path = match[1];
      parts.push(
        <a
          key={`${match.index}-${path}`}
          href="#"
          onClick={(e) => {
            e.preventDefault();
            navigateToPath(path);
          }}
          style={{ color: '#4F46E5', textDecoration: 'underline', cursor: 'pointer' }}
          title={`点击导航到: ${path}`}
        >
          {path}
        </a>
      );

      lastIndex = navRegex.lastIndex;
    }

    // 添加最后剩余的文本
    if (lastIndex < text.length) {
      parts.push(text.substring(lastIndex));
    }

    return parts.length > 1 ? <>{parts}</> : text;
  };

  const renderActions = (actions = []) => {
    if (!actions.length) {
      return null;
    }

    return (
      <div className="ai-chatbot-response-section">
        <div className="ai-chatbot-response-label">快捷操作</div>
        <div className="ai-chatbot-action-list">
          {actions.map((action, index) => (
            <button
              key={`${action.type}-${action.path || index}`}
              type="button"
              onClick={() => navigateToPath(action.path)}
              className="ai-chatbot-action-chip"
            >
              {action.label}
            </button>
          ))}
        </div>
      </div>
    );
  };

  const renderCitations = (message) => {
    const citations = message?.citations || [];
    if (!shouldShowCitations(message)) {
      return null;
    }

    return (
      <div className="ai-chatbot-response-section">
        <details className="ai-chatbot-citation-details">
          <summary className="ai-chatbot-response-label ai-chatbot-citation-summary">
            参考文档 ({citations.length})
          </summary>
          <div className="ai-chatbot-citation-list">
            {citations.map((citation, index) => (
              <div key={`${citation.path || citation.title}-${index}`} className="ai-chatbot-citation-card">
                <div className="ai-chatbot-citation-title">{citation.title}</div>
                {citation.path ? (
                  <div className="ai-chatbot-citation-path">{citation.path}</div>
                ) : null}
                {citation.snippet ? (
                  <div className="ai-chatbot-citation-snippet">{citation.snippet}</div>
                ) : null}
              </div>
            ))}
          </div>
        </details>
      </div>
    );
  };

  const renderAIResponse = (message) => {
    const { summary, items } = parseAssistantSummary(message.text);
    const resultCards = message.resultCards || [];
    const fallbackItems = resultCards.length ? [] : items;

    return (
      <>
        {shouldShowIntentBadge(message.intent) ? (
          <div className="ai-chatbot-badge-row">
            {shouldShowIntentBadge(message.intent) ? (
              <div className="ai-chatbot-intent-badge">{formatIntentLabel(message.intent)}</div>
            ) : null}
          </div>
        ) : null}
        {summary ? (
          <div className="ai-chatbot-text">
            {renderMessageText(summary)}
          </div>
        ) : null}
        {(resultCards.length || fallbackItems.length) ? (
          <div className="ai-chatbot-response-section">
            <div className="ai-chatbot-response-label">查询结果</div>
            <div className="ai-chatbot-result-list">
              {resultCards.map((card, index) => (
                <div
                  key={`${message.id}-card-${index}`}
                  className={`ai-chatbot-result-card ai-chatbot-result-card-${card.sourceType || 'default'}`}
                >
                  {card.sourceType || card.toolName ? (
                    <div className="ai-chatbot-result-pill-row">
                      {card.sourceType ? (
                        <span className={`ai-chatbot-result-pill ai-chatbot-result-pill-${card.sourceType}`}>{card.sourceType}</span>
                      ) : null}
                      {card.toolName ? (
                        <span className="ai-chatbot-result-pill ai-chatbot-result-pill-tool">{card.toolName}</span>
                      ) : null}
                    </div>
                  ) : null}
                  <div className="ai-chatbot-result-title">{card.title}</div>
                  {card.subtitle ? (
                    <div className="ai-chatbot-result-subtitle">{card.subtitle}</div>
                  ) : null}
                  {card.meta ? (
                    <div className="ai-chatbot-result-meta">{card.meta}</div>
                  ) : null}
                </div>
              ))}
              {fallbackItems.map((item, index) => (
                <div key={`${message.id}-item-${index}`} className="ai-chatbot-result-card">
                  <div className="ai-chatbot-result-title">{item}</div>
                </div>
              ))}
            </div>
          </div>
        ) : null}
        {renderActions(message.actions)}
        {renderCitations(message)}
      </>
    );
  };

  return (
    <>
      {/* 浮动按钮 */}
      <div className="ai-chatbot-float-button" onClick={toggleChat}>
        <div className="ai-chatbot-icon"></div>
        {unreadCount > 0 && (
          <span className="ai-chatbot-unread-badge">{unreadCount}</span>
        )}
      </div>

      {/* 聊天窗口 */}
      {isOpen && (
        <div className={`ai-chatbot-container ${isMinimized ? 'minimized' : ''} ${isMaximized ? 'maximized' : ''}`}>
          {/* 顶部栏 */}
          <div className="ai-chatbot-header">
            <div className="ai-chatbot-title-group">
              <div className="ai-chatbot-title-icon"></div>
              <div className="ai-chatbot-title">运维小助手</div>
            </div>
            <div className="ai-chatbot-controls">
              <button
                type="button"
                className="ai-chatbot-session-btn"
                onClick={(e) => {
                  e.stopPropagation();
                  setSessionPanelOpen((open) => !open);
                }}
                title="会话历史"
              >
                ≡
              </button>
              <button
                type="button"
                className="ai-chatbot-session-btn"
                onClick={(e) => {
                  e.stopPropagation();
                  handleCreateSession();
                }}
                title="新建会话"
              >
                ＋
              </button>
              <button
                type="button"
                className="ai-chatbot-maximize-btn"
                onClick={(e) => {
                  e.stopPropagation();
                  toggleMaximize();
                }}
                title={isMaximized ? "恢复" : "最大化"}
              >
                {isMaximized ? '❐' : '☐'}
              </button>
              <button
                type="button"
                className="ai-chatbot-minimize-btn"
                onClick={(e) => {
                  e.stopPropagation();
                  toggleMinimize();
                }}
              >
                {isMinimized ? '+' : '−'}
              </button>
              <button
                type="button"
                className="ai-chatbot-close-btn"
                onClick={(e) => {
                  e.stopPropagation();
                  closeChat();
                }}
              >
                ×
              </button>
            </div>
          </div>

          {/* 聊天内容区域 */}
          {!isMinimized && (
            <div className="ai-chatbot-content">
              {sessionPanelOpen && (
                <aside className="ai-chatbot-session-panel">
                  <div className="ai-chatbot-session-panel-header">
                    <span>最近会话</span>
                    <button
                      type="button"
                      className="ai-chatbot-session-link"
                      onClick={handleCreateSession}
                    >
                      新建会话
                    </button>
                  </div>
                  <div className="ai-chatbot-session-list">
                    <div className="ai-chatbot-session-filters">
                      <button
                        type="button"
                        className={`ai-chatbot-session-filter ${sessionStatus === 'active' ? 'active' : ''}`}
                        onClick={() => handleSessionStatusChange('active')}
                      >
                        进行中
                      </button>
                      <button
                        type="button"
                        className={`ai-chatbot-session-filter ${sessionStatus === 'archived' ? 'active' : ''}`}
                        onClick={() => handleSessionStatusChange('archived')}
                      >
                        已归档
                      </button>
                      <button
                        type="button"
                        className={`ai-chatbot-session-filter ${sessionStatus === 'all' ? 'active' : ''}`}
                        onClick={() => handleSessionStatusChange('all')}
                      >
                        全部
                      </button>
                    </div>
                    <div style={{ padding: '8px 12px' }}>
                      <input
                        type="search"
                        value={sessionQuery}
                        onChange={handleSessionSearch}
                        placeholder="搜索会话标题或摘要"
                        style={{
                          width: '100%',
                          borderRadius: '8px',
                          border: '1px solid #d9d9d9',
                          padding: '8px 10px',
                          fontSize: '12px'
                        }}
                      />
                    </div>
                    {sessionsLoading ? (
                      <div className="ai-chatbot-session-empty">会话加载中...</div>
                    ) : sessions.length === 0 ? (
                      <div className="ai-chatbot-session-empty">暂无历史会话</div>
                    ) : (
                      sessions.map((session) => (
                        <div
                          key={session.sessionId}
                          className={`ai-chatbot-session-item ${session.sessionId === sessionId ? 'active' : ''}`}
                        >
                          <button
                            type="button"
                            className="ai-chatbot-session-main"
                            onClick={() => handleSelectSession(session.sessionId)}
                          >
                            <div className="ai-chatbot-session-summary">
                              {session.pinned ? '置顶 · ' : ''}
                              {session.title || session.summary || '新会话'}
                            </div>
                            <div className="ai-chatbot-session-meta">
                              <span>{session.status === 'active' ? '进行中' : '历史会话'}</span>
                              <span>{session.messageCount ? `${session.messageCount} 条` : formatSessionTime(session.updatedAt)}</span>
                            </div>
                          </button>
                          <div className="ai-chatbot-session-actions">
                            <button
                              type="button"
                              className="ai-chatbot-session-link"
                              onClick={() => handlePinSession(session)}
                            >
                              {session.pinned ? '取消固定' : '固定'}
                            </button>
                            <button
                              type="button"
                              className="ai-chatbot-session-link"
                              disabled={renamingSessionId === session.sessionId}
                              onClick={() => handleRenameSession(session)}
                            >
                              重命名
                            </button>
                            {session.status !== 'archived' ? (
                              <button
                                type="button"
                                className="ai-chatbot-session-link"
                                onClick={() => handleArchiveSession(session.sessionId)}
                              >
                                归档
                              </button>
                            ) : (
                              <button
                                type="button"
                                className="ai-chatbot-session-link"
                                onClick={() => updateSession(session.sessionId, { status: 'active' }).then(() => fetchSessions({ page: 1, query: sessionQuery, status: sessionStatus, append: false })).catch((error) => {
                                  console.error('恢复会话失败:', error);
                                  appendSystemMessage('恢复会话失败，请稍后重试。');
                                })}
                              >
                                恢复
                              </button>
                            )}
                            <button
                              type="button"
                              className="ai-chatbot-session-link"
                              onClick={() => handleDeleteSession(session.sessionId)}
                            >
                              删除
                            </button>
                          </div>
                        </div>
                      ))
                    )}
                    {!sessionsLoading && sessionHasMore ? (
                      <div style={{ padding: '8px 12px 12px' }}>
                        <button
                          type="button"
                          className="ai-chatbot-session-link"
                          onClick={handleLoadMoreSessions}
                          style={{ width: '100%', justifyContent: 'center' }}
                        >
                          加载更多
                        </button>
                      </div>
                    ) : null}
                  </div>
                </aside>
              )}
              <div className="ai-chatbot-main">
                <div className="ai-chatbot-messages">
                  {messageHasMore && sessionId ? (
                    <button
                      type="button"
                      className="ai-chatbot-load-more"
                      onClick={handleLoadOlderMessages}
                    >
                      加载更早消息
                    </button>
                  ) : null}
                  {messages.length === 0 ? (
                    <div className="ai-chatbot-welcome">
                      <p>您好！我是运维小助手。</p>
                      <p>目前支持基础问答、页面导航和只读查询。</p>
                      <p>你可以继续问我部署、监控、告警、安全或系统管理相关问题。</p>
                    </div>
                  ) : (
                    messages.map((message) => (
                      <div
                        key={message.id}
                        className={`ai-chatbot-message ${message.sender}-message`}
                      >
                        <div className="ai-chatbot-avatar">
                          {message.sender === 'user' ? '👤' : '🤖'}
                        </div>
                        <div className="ai-chatbot-message-content">
                          {message.sender === 'ai' ? renderAIResponse(message) : (
                            <div className="ai-chatbot-text">
                              {renderMessageText(message.text)}
                            </div>
                          )}
                          <div className="ai-chatbot-timestamp">{message.timestamp}</div>
                        </div>
                      </div>
                    ))
                  )}
                  {isLoading && (
                    <div className="ai-chatbot-message ai-message">
                      <div className="ai-chatbot-avatar">🤖</div>
                      <div className="ai-chatbot-message-content">
                        <div className="ai-chatbot-typing-indicator">
                          <span></span>
                          <span></span>
                          <span></span>
                        </div>
                      </div>
                    </div>
                  )}
                  <div ref={messagesEndRef} />
                </div>

                {/* 输入区域 */}
                <div className="ai-chatbot-input-area">
                  <textarea
                    value={inputValue}
                    onChange={(e) => setInputValue(e.target.value)}
                    onKeyPress={handleKeyPress}
                    placeholder="输入消息..."
                    className="ai-chatbot-input"
                    rows="1"
                  />
                  <button
                    onClick={sendMessage}
                    disabled={!inputValue.trim() || isLoading}
                    className="ai-chatbot-send-btn"
                  >
                    发送
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
      {isMaximized && isOpen && !isMinimized && (
        <div
          className="ai-chatbot-overlay"
          onClick={() => setIsMaximized(false)}
        />
      )}
    </>
  );
};

export default AIChatbot;