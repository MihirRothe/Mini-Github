import React, { useState, useEffect, useRef } from 'react';
import { useLocation } from 'react-router-dom';
import {
  Sparkles,
  X,
  Send,
  Bot,
  User,
  Copy,
  Check,
  Loader2,
  RefreshCw,
  Cpu,
  Code,
  ShieldCheck,
  FileQuestion,
  Lightbulb
} from 'lucide-react';

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
}

interface ProviderInfo {
  id: string;
  name: string;
  active: boolean;
  configured: boolean;
  model: string;
  description: string;
}

interface ForgeAIDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ForgeAIDrawer: React.FC<ForgeAIDrawerProps> = ({ isOpen, onClose }) => {
  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      id: 'welcome',
      role: 'assistant',
      content:
        'Hello! I am **ForgeAI**, your developer collaborator. I can help you understand this codebase, conduct security reviews on pull requests, generate comprehensive unit tests, and answer architectural questions. How can I assist you today?',
      timestamp: 'Just now',
    },
  ]);
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(false);
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [selectedProvider, setSelectedProvider] = useState<string>('local');
  const [activeProviderName, setActiveProviderName] = useState<string>('Local Engine');
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const location = useLocation();
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Derive contextual repository or file information from location
  const pathParts = location.pathname.split('/').filter(Boolean);
  const isRepoContext = pathParts.length >= 2 && !['dashboard', 'explore', 'status', 'search', 'login', 'register', 'settings', 'orgs'].includes(pathParts[0]);
  const repoOwner = isRepoContext ? pathParts[0] : null;
  const repoName = isRepoContext ? pathParts[1] : null;

  // Fetch available AI providers
  useEffect(() => {
    fetch('/api/v1/ai/providers', { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data?.providers) {
          setProviders(data.providers);
          if (data.active_provider) {
            setSelectedProvider(data.active_provider);
            const active = data.providers.find((p: ProviderInfo) => p.id === data.active_provider);
            if (active) setActiveProviderName(active.name);
          }
        }
      })
      .catch(() => {});
  }, []);

  // Auto-scroll to bottom of messages
  useEffect(() => {
    if (isOpen) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
      inputRef.current?.focus();
    }
  }, [messages, isOpen]);

  // Keyboard shortcut listener (Ctrl+Shift+I or Escape)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  const handleSendMessage = async (customPrompt?: string) => {
    const textToSend = customPrompt || inputValue.trim();
    if (!textToSend || loading) return;

    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      role: 'user',
      content: textToSend,
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    };

    const newMessages = [...messages, userMessage];
    setMessages(newMessages);
    if (!customPrompt) setInputValue('');
    setLoading(true);

    try {
      const apiMessages = newMessages
        .filter((m) => m.id !== 'welcome')
        .map((m) => ({ role: m.role, content: m.content }));

      const res = await fetch(`/api/v1/ai/chat?provider=${selectedProvider}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          messages: apiMessages,
          repository: repoOwner && repoName ? `${repoOwner}/${repoName}` : undefined,
        }),
      });

      if (!res.ok) {
        throw new Error('Failed to reach ForgeAI service.');
      }

      const data = await res.json();
      const assistantMessage: ChatMessage = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: data.message?.content || 'I processed your request, but received an empty response.',
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      };

      setMessages((prev) => [...prev, assistantMessage]);
    } catch (err: any) {
      const errorMessage: ChatMessage = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: `⚠️ **Error**: ${err.message || 'Unable to complete AI inference.'}`,
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      };
      setMessages((prev) => [...prev, errorMessage]);
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = (id: string, text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const handleClearHistory = () => {
    setMessages([
      {
        id: 'welcome',
        role: 'assistant',
        content: 'Conversation history cleared. How can I assist you?',
        timestamp: 'Just now',
      },
    ]);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-hidden pointer-events-none animate-fade-in">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-sm pointer-events-auto transition-opacity"
        onClick={onClose}
      />

      {/* Slide-over Drawer */}
      <div className="absolute inset-y-0 right-0 max-w-full flex pl-10 pointer-events-auto">
        <div className="w-screen max-w-md md:max-w-lg bg-forge-surface border-l border-forge-border shadow-2xl flex flex-col">
          {/* Drawer Header */}
          <div className="p-4 border-b border-forge-border bg-forge-card/80 flex items-center justify-between">
            <div className="flex items-center space-x-2.5">
              <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-purple-600 to-indigo-500 flex items-center justify-center shadow-lg shadow-purple-500/20 text-white">
                <Sparkles className="w-4 h-4" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-bold text-white tracking-tight">ForgeAI Assistant</h3>
                  <span className="text-[10px] font-mono px-1.5 py-0.2 bg-purple-500/10 text-purple-400 border border-purple-500/20 rounded">
                    v1.0
                  </span>
                </div>
                <div className="flex items-center gap-1.5 text-[11px] text-forge-muted">
                  <Cpu className="w-3 h-3 text-emerald-400" />
                  <span>{activeProviderName}</span>
                </div>
              </div>
            </div>

            <div className="flex items-center space-x-1.5">
              <button
                onClick={handleClearHistory}
                title="Clear conversation"
                className="p-1.5 text-forge-muted hover:text-white rounded-md hover:bg-forge-bg transition-colors"
              >
                <RefreshCw className="w-4 h-4" />
              </button>
              <button
                onClick={onClose}
                title="Close drawer (Esc)"
                className="p-1.5 text-forge-muted hover:text-white rounded-md hover:bg-forge-bg transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Context Banner */}
          {repoOwner && repoName && (
            <div className="px-4 py-2 bg-purple-950/20 border-b border-purple-500/10 flex items-center justify-between text-xs">
              <div className="flex items-center gap-1.5 text-purple-300 truncate">
                <Code className="w-3.5 h-3.5 flex-shrink-0" />
                <span className="font-mono truncate">{repoOwner}/{repoName}</span>
              </div>
              <span className="text-[10px] text-purple-400/70 uppercase tracking-wider font-semibold">Active Repo</span>
            </div>
          )}

          {/* Messages Area */}
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {messages.map((msg) => (
              <div
                key={msg.id}
                className={`flex gap-3 text-xs leading-relaxed ${
                  msg.role === 'user' ? 'justify-end' : 'justify-start'
                }`}
              >
                {msg.role === 'assistant' && (
                  <div className="w-7 h-7 rounded-lg bg-purple-600/20 border border-purple-500/30 flex items-center justify-center text-purple-300 flex-shrink-0 mt-0.5">
                    <Bot className="w-3.5 h-3.5" />
                  </div>
                )}

                <div
                  className={`relative group max-w-[85%] rounded-xl p-3.5 shadow-sm ${
                    msg.role === 'user'
                      ? 'bg-blue-600 text-white rounded-tr-none'
                      : 'bg-forge-card border border-forge-border text-forge-text rounded-tl-none'
                  }`}
                >
                  {/* Markdown-like formatting simple render */}
                  <div className="whitespace-pre-wrap font-sans">
                    {msg.content}
                  </div>

                  <div className="mt-2 flex items-center justify-between text-[10px] text-forge-muted/70 pt-1 border-t border-forge-border/40">
                    <span>{msg.timestamp}</span>
                    <button
                      onClick={() => handleCopy(msg.id, msg.content)}
                      className="opacity-0 group-hover:opacity-100 transition-opacity text-forge-muted hover:text-white flex items-center gap-1"
                    >
                      {copiedId === msg.id ? (
                        <>
                          <Check className="w-3 h-3 text-emerald-400" />
                          <span>Copied</span>
                        </>
                      ) : (
                        <>
                          <Copy className="w-3 h-3" />
                          <span>Copy</span>
                        </>
                      )}
                    </button>
                  </div>
                </div>

                {msg.role === 'user' && (
                  <div className="w-7 h-7 rounded-lg bg-blue-600/20 border border-blue-500/30 flex items-center justify-center text-blue-300 flex-shrink-0 mt-0.5">
                    <User className="w-3.5 h-3.5" />
                  </div>
                )}
              </div>
            ))}

            {loading && (
              <div className="flex gap-3 text-xs justify-start">
                <div className="w-7 h-7 rounded-lg bg-purple-600/20 border border-purple-500/30 flex items-center justify-center text-purple-300 flex-shrink-0">
                  <Bot className="w-3.5 h-3.5 animate-pulse" />
                </div>
                <div className="bg-forge-card border border-forge-border text-forge-muted rounded-xl rounded-tl-none p-3.5 flex items-center gap-2">
                  <Loader2 className="w-3.5 h-3.5 animate-spin text-purple-400" />
                  <span>ForgeAI is analyzing and generating response...</span>
                </div>
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          {/* Quick Action Prompt Chips */}
          <div className="px-4 py-2 border-t border-forge-border bg-forge-card/40 flex items-center gap-2 overflow-x-auto text-xs no-scrollbar">
            <button
              onClick={() => handleSendMessage('Explain the architecture and main packages of ForgeHub')}
              disabled={loading}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-forge-bg border border-forge-border hover:border-purple-500/40 text-forge-muted hover:text-purple-300 whitespace-nowrap transition-colors"
            >
              <Lightbulb className="w-3 h-3 text-amber-400" />
              <span>Architecture</span>
            </button>
            <button
              onClick={() => handleSendMessage('How does Git Smart HTTP push/pull authentication work in this platform?')}
              disabled={loading}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-forge-bg border border-forge-border hover:border-purple-500/40 text-forge-muted hover:text-purple-300 whitespace-nowrap transition-colors"
            >
              <ShieldCheck className="w-3 h-3 text-emerald-400" />
              <span>Git HTTP Engine</span>
            </button>
            <button
              onClick={() => handleSendMessage('What security and injection protections does ForgeAI enforce?')}
              disabled={loading}
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-forge-bg border border-forge-border hover:border-purple-500/40 text-forge-muted hover:text-purple-300 whitespace-nowrap transition-colors"
            >
              <FileQuestion className="w-3 h-3 text-blue-400" />
              <span>AI Security</span>
            </button>
          </div>

          {/* Input Area */}
          <div className="p-4 border-t border-forge-border bg-forge-card/80">
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleSendMessage();
              }}
              className="flex flex-col gap-2"
            >
              <div className="relative">
                <textarea
                  ref={inputRef}
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault();
                      handleSendMessage();
                    }
                  }}
                  placeholder="Ask ForgeAI anything about your code, diffs, or architecture..."
                  rows={2}
                  disabled={loading}
                  className="w-full px-3 py-2 pr-10 rounded-xl bg-forge-bg border border-forge-border focus:border-purple-500 focus:ring-1 focus:ring-purple-500 text-xs text-forge-text resize-none transition-all placeholder:text-forge-muted/60"
                />
                <button
                  type="submit"
                  disabled={loading || !inputValue.trim()}
                  className="absolute right-2.5 bottom-3 p-1.5 rounded-lg bg-purple-600 hover:bg-purple-500 disabled:opacity-40 disabled:hover:bg-purple-600 text-white transition-colors shadow-md shadow-purple-600/20"
                >
                  <Send className="w-3.5 h-3.5" />
                </button>
              </div>

              <div className="flex items-center justify-between text-[11px] text-forge-muted">
                <div className="flex items-center gap-1">
                  <span>Provider:</span>
                  <select
                    value={selectedProvider}
                    onChange={(e) => {
                      setSelectedProvider(e.target.value);
                      const p = providers.find((prov) => prov.id === e.target.value);
                      if (p) setActiveProviderName(p.name);
                    }}
                    className="bg-transparent text-forge-text border-none focus:outline-none cursor-pointer"
                  >
                    {providers.map((p) => (
                      <option key={p.id} value={p.id} className="bg-forge-surface text-forge-text">
                        {p.name} {p.active ? '(Default)' : ''}
                      </option>
                    ))}
                  </select>
                </div>
                <span>Shift+Enter for newline</span>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
};
