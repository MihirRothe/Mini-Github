import React, { useState, useEffect } from 'react';
import {
  Sparkles,
  X,
  FileCode,
  Copy,
  Check,
  Loader2,
  AlertTriangle,
  Lightbulb,
  Clock,
  HardDrive,
  Shield,
  Layers,
  TestTube
} from 'lucide-react';

interface ExplainResponse {
  summary: string;
  language: string;
  key_components: string[];
  complexity: {
    time: string;
    space: string;
    description: string;
  };
  architectural_role: string;
  potential_risks: string[];
  suggestions: string[];
  raw_markdown?: string;
}

interface GenerateTestsResponse {
  language: string;
  framework: string;
  test_code: string;
  test_scenarios: Array<{
    name: string;
    description: string;
    type: string;
  }>;
  instructions: string;
}

interface AIExplainModalProps {
  isOpen: boolean;
  onClose: () => void;
  fileName: string;
  code: string;
  initialMode?: 'explain' | 'tests';
}

export const AIExplainModal: React.FC<AIExplainModalProps> = ({
  isOpen,
  onClose,
  fileName,
  code,
  initialMode = 'explain',
}) => {
  const [activeTab, setActiveTab] = useState<'explain' | 'tests'>(initialMode);
  const [loading, setLoading] = useState(false);
  const [explanation, setExplanation] = useState<ExplainResponse | null>(null);
  const [tests, setTests] = useState<GenerateTestsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setActiveTab(initialMode);
      if (initialMode === 'explain' && !explanation) {
        fetchExplanation();
      } else if (initialMode === 'tests' && !tests) {
        fetchTests();
      }
    }
  }, [isOpen, initialMode]);

  const fetchExplanation = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/v1/ai/explain', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          code,
          file_path: fileName,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to explain code snippet.');
      }
      setExplanation(data.explanation);
    } catch (err: any) {
      setError(err.message || 'Error running explanation');
    } finally {
      setLoading(false);
    }
  };

  const fetchTests = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/v1/ai/generate-tests', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          code,
          file_path: fileName,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to generate unit tests.');
      }
      setTests(data.test_suite);
    } catch (err: any) {
      setError(err.message || 'Error generating test suite');
    } finally {
      setLoading(false);
    }
  };

  const handleTabChange = (tab: 'explain' | 'tests') => {
    setActiveTab(tab);
    if (tab === 'explain' && !explanation && !loading) {
      fetchExplanation();
    } else if (tab === 'tests' && !tests && !loading) {
      fetchTests();
    }
  };

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
      <div className="bg-forge-surface border border-forge-border w-full max-w-3xl max-h-[85vh] rounded-2xl shadow-2xl flex flex-col overflow-hidden">
        {/* Modal Header */}
        <div className="p-5 border-b border-forge-border bg-forge-card/80 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-purple-600 to-indigo-500 flex items-center justify-center shadow-lg shadow-purple-500/20 text-white">
              <Sparkles className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-white flex items-center gap-2">
                ForgeAI Code Assistant
                <span className="text-[10px] font-mono px-2 py-0.5 bg-purple-500/10 text-purple-400 border border-purple-500/20 rounded">
                  {fileName}
                </span>
              </h2>
              <p className="text-xs text-forge-muted">
                Architectural breakdown, complexity calculation, and unit test generation
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 text-forge-muted hover:text-white rounded-lg hover:bg-forge-bg transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-forge-border bg-forge-card/40 px-6">
          <button
            onClick={() => handleTabChange('explain')}
            className={`py-3 px-4 text-xs font-semibold border-b-2 transition-colors flex items-center gap-2 ${
              activeTab === 'explain'
                ? 'border-purple-500 text-purple-400'
                : 'border-transparent text-forge-muted hover:text-white'
            }`}
          >
            <Layers className="w-4 h-4" />
            <span>Code Explanation</span>
          </button>
          <button
            onClick={() => handleTabChange('tests')}
            className={`py-3 px-4 text-xs font-semibold border-b-2 transition-colors flex items-center gap-2 ${
              activeTab === 'tests'
                ? 'border-purple-500 text-purple-400'
                : 'border-transparent text-forge-muted hover:text-white'
            }`}
          >
            <TestTube className="w-4 h-4" />
            <span>Generate Unit Tests</span>
          </button>
        </div>

        {/* Modal Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {loading && (
            <div className="py-16 text-center space-y-3">
              <Loader2 className="w-8 h-8 text-purple-400 animate-spin mx-auto" />
              <div className="space-y-1">
                <h4 className="text-sm font-semibold text-white">
                  {activeTab === 'explain' ? 'Analyzing Code Structure...' : 'Synthesizing Test Suite...'}
                </h4>
                <p className="text-xs text-forge-muted">
                  ForgeAI is inspecting symbols, evaluating cyclomatic flow, and generating output.
                </p>
              </div>
            </div>
          )}

          {error && (
            <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs flex items-start gap-3">
              <AlertTriangle className="w-5 h-5 flex-shrink-0 mt-0.5" />
              <div>
                <div className="font-semibold">Analysis Failed</div>
                <div>{error}</div>
              </div>
            </div>
          )}

          {!loading && !error && activeTab === 'explain' && explanation && (
            <div className="space-y-5 text-xs">
              {/* Summary */}
              <div className="p-4 rounded-xl bg-forge-card border border-forge-border space-y-2">
                <div className="flex items-center justify-between">
                  <h4 className="font-semibold text-white flex items-center gap-2">
                    <Lightbulb className="w-4 h-4 text-amber-400" />
                    Overview
                  </h4>
                  <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-purple-500/10 text-purple-300 border border-purple-500/20">
                    {explanation.language}
                  </span>
                </div>
                <p className="text-forge-text leading-relaxed">{explanation.summary}</p>
              </div>

              {/* Key Components */}
              {explanation.key_components?.length > 0 && (
                <div className="space-y-2">
                  <h4 className="text-[11px] font-semibold uppercase tracking-wider text-forge-muted">
                    Key Components & Declarations
                  </h4>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    {explanation.key_components.map((comp, i) => (
                      <div
                        key={i}
                        className="p-2.5 rounded-lg bg-forge-bg border border-forge-border text-forge-text font-mono text-[11px] flex items-center gap-2"
                      >
                        <FileCode className="w-3.5 h-3.5 text-purple-400 flex-shrink-0" />
                        <span className="truncate">{comp}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Complexity Analysis */}
              {explanation.complexity && (
                <div className="p-4 rounded-xl bg-forge-card border border-forge-border space-y-3">
                  <h4 className="font-semibold text-white">Computational Complexity</h4>
                  <div className="grid grid-cols-2 gap-3">
                    <div className="p-2.5 rounded-lg bg-forge-bg border border-forge-border flex items-center gap-2.5">
                      <Clock className="w-4 h-4 text-blue-400" />
                      <div>
                        <div className="text-[10px] text-forge-muted">Time Complexity</div>
                        <div className="font-mono font-bold text-white">{explanation.complexity.time}</div>
                      </div>
                    </div>
                    <div className="p-2.5 rounded-lg bg-forge-bg border border-forge-border flex items-center gap-2.5">
                      <HardDrive className="w-4 h-4 text-emerald-400" />
                      <div>
                        <div className="text-[10px] text-forge-muted">Space Complexity</div>
                        <div className="font-mono font-bold text-white">{explanation.complexity.space}</div>
                      </div>
                    </div>
                  </div>
                  <p className="text-forge-muted text-[11px]">{explanation.complexity.description}</p>
                </div>
              )}

              {/* Risks & Suggestions */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {explanation.potential_risks?.length > 0 && (
                  <div className="p-3.5 rounded-xl bg-amber-500/5 border border-amber-500/20 space-y-2">
                    <h4 className="font-semibold text-amber-400 flex items-center gap-1.5 text-xs">
                      <Shield className="w-3.5 h-3.5" />
                      Caveats & Risks
                    </h4>
                    <ul className="space-y-1.5 text-forge-text text-[11px]">
                      {explanation.potential_risks.map((r, i) => (
                        <li key={i} className="flex items-start gap-1.5">
                          <span className="text-amber-400 mt-0.5">•</span>
                          <span>{r}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {explanation.suggestions?.length > 0 && (
                  <div className="p-3.5 rounded-xl bg-purple-500/5 border border-purple-500/20 space-y-2">
                    <h4 className="font-semibold text-purple-400 flex items-center gap-1.5 text-xs">
                      <Sparkles className="w-3.5 h-3.5" />
                      Recommendations
                    </h4>
                    <ul className="space-y-1.5 text-forge-text text-[11px]">
                      {explanation.suggestions.map((s, i) => (
                        <li key={i} className="flex items-start gap-1.5">
                          <span className="text-purple-400 mt-0.5">•</span>
                          <span>{s}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            </div>
          )}

          {!loading && !error && activeTab === 'tests' && tests && (
            <div className="space-y-5 text-xs">
              {/* Test Header */}
              <div className="flex items-center justify-between p-3 rounded-xl bg-forge-card border border-forge-border">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-white">Framework:</span>
                  <span className="px-2 py-0.5 rounded font-mono bg-purple-500/10 text-purple-300 border border-purple-500/20">
                    {tests.framework} ({tests.language})
                  </span>
                </div>
                <button
                  onClick={() => handleCopy(tests.test_code)}
                  className="px-3 py-1 rounded-lg bg-forge-bg hover:bg-forge-border border border-forge-border text-forge-text flex items-center gap-1.5 transition-colors"
                >
                  {copied ? (
                    <>
                      <Check className="w-3.5 h-3.5 text-emerald-400" />
                      <span>Copied</span>
                    </>
                  ) : (
                    <>
                      <Copy className="w-3.5 h-3.5" />
                      <span>Copy Test Code</span>
                    </>
                  )}
                </button>
              </div>

              {/* Scenarios Covered */}
              {tests.test_scenarios?.length > 0 && (
                <div className="space-y-2">
                  <h4 className="text-[11px] font-semibold uppercase tracking-wider text-forge-muted">
                    Test Scenarios ({tests.test_scenarios.length})
                  </h4>
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                    {tests.test_scenarios.map((sc, i) => (
                      <div key={i} className="p-2.5 rounded-lg bg-forge-card border border-forge-border space-y-1">
                        <div className="flex items-center justify-between">
                          <span className="font-semibold text-white truncate">{sc.name}</span>
                          <span className="px-1.5 py-0.2 rounded text-[9px] font-mono uppercase bg-forge-bg text-purple-400 border border-forge-border">
                            {sc.type}
                          </span>
                        </div>
                        <p className="text-forge-muted text-[10px] leading-relaxed">{sc.description}</p>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Code Editor Preview */}
              <div className="space-y-2">
                <h4 className="text-[11px] font-semibold uppercase tracking-wider text-forge-muted">
                  Generated Suite
                </h4>
                <pre className="p-4 rounded-xl bg-forge-bg border border-forge-border overflow-x-auto text-[11px] font-mono text-forge-text leading-relaxed">
                  {tests.test_code}
                </pre>
              </div>

              {/* Instructions */}
              {tests.instructions && (
                <div className="p-3 rounded-lg bg-blue-500/5 border border-blue-500/20 text-blue-300 text-[11px]">
                  💡 {tests.instructions}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
