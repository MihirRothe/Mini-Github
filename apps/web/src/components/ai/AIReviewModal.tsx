import React, { useState } from 'react';
import {
  Sparkles,
  X,
  ShieldAlert,
  CheckCircle,
  MessageSquare,
  Copy,
  Check,
  Loader2,
  FileCode,
  Send,
  Wand2
} from 'lucide-react';

export interface ReviewFinding {
  file: string;
  line?: number;
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'INFO';
  category: 'SECURITY' | 'BUG' | 'PERFORMANCE' | 'STYLE' | 'ARCHITECTURE';
  title: string;
  description: string;
  suggested_fix?: string;
}

export interface ReviewResponse {
  verdict: 'APPROVE' | 'REQUEST_CHANGES' | 'COMMENT';
  summary: string;
  score: number;
  findings: ReviewFinding[];
  review_comment: string;
}

interface AIReviewModalProps {
  isOpen: boolean;
  onClose: () => void;
  owner: string;
  repo: string;
  pullNumber: number;
  onReviewPosted?: () => void;
}

export const AIReviewModal: React.FC<AIReviewModalProps> = ({
  isOpen,
  onClose,
  owner,
  repo,
  pullNumber,
  onReviewPosted,
}) => {
  const [loading, setLoading] = useState(false);
  const [posting, setPosting] = useState(false);
  const [review, setReview] = useState<ReviewResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [postedSuccess, setPostedSuccess] = useState(false);

  const runReview = async (autoPost = false) => {
    setLoading(true);
    setError(null);
    setPostedSuccess(false);

    try {
      const res = await fetch(
        `/api/v1/repos/${owner}/${repo}/pulls/${pullNumber}/ai-review${autoPost ? '?post_comment=true' : ''}`,
        {
          method: 'POST',
          credentials: 'include',
        }
      );

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to complete AI review.');
      }

      setReview(data.review);
      if (autoPost) {
        setPostedSuccess(true);
        if (onReviewPosted) onReviewPosted();
      }
    } catch (err: any) {
      setError(err.message || 'Error running automated review.');
    } finally {
      setLoading(false);
    }
  };

  const handlePostReview = async () => {
    if (!review) return;
    setPosting(true);
    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repo}/pulls/${pullNumber}/reviews`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          state:
            review.verdict === 'APPROVE'
              ? 'approved'
              : review.verdict === 'REQUEST_CHANGES'
              ? 'changes_requested'
              : 'commented',
          body: review.review_comment,
        }),
      });

      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error?.message || 'Failed to post review');
      }

      setPostedSuccess(true);
      if (onReviewPosted) onReviewPosted();
    } catch (err: any) {
      setError(err.message || 'Error publishing review comment');
    } finally {
      setPosting(false);
    }
  };

  const handleCopy = () => {
    if (!review) return;
    navigator.clipboard.writeText(review.review_comment);
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
                ForgeAI Code Review Assistant
                <span className="text-[10px] font-mono px-2 py-0.5 bg-purple-500/10 text-purple-400 border border-purple-500/20 rounded">
                  PR #{pullNumber}
                </span>
              </h2>
              <p className="text-xs text-forge-muted">
                Automated security auditing, correctness analysis, and quality assessment
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

        {/* Modal Body */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {!review && !loading && !error && (
            <div className="py-12 text-center space-y-4">
              <div className="w-16 h-16 rounded-2xl bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-400 mx-auto">
                <Wand2 className="w-8 h-8" />
              </div>
              <div className="max-w-md mx-auto space-y-1">
                <h3 className="text-sm font-semibold text-white">Analyze Pull Request Changeset</h3>
                <p className="text-xs text-forge-muted leading-relaxed">
                  ForgeAI will inspect the full diff, verify security constraints, check for memory or concurrency leaks, and generate structured feedback.
                </p>
              </div>
              <button
                onClick={() => runReview(false)}
                className="px-4 py-2 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-semibold shadow-lg shadow-purple-600/20 transition-all inline-flex items-center gap-2"
              >
                <Sparkles className="w-4 h-4" />
                <span>Start AI Review</span>
              </button>
            </div>
          )}

          {loading && (
            <div className="py-16 text-center space-y-3">
              <Loader2 className="w-8 h-8 text-purple-400 animate-spin mx-auto" />
              <div className="space-y-1">
                <h4 className="text-sm font-semibold text-white">Reviewing Diff...</h4>
                <p className="text-xs text-forge-muted">
                  Inspecting modified files, evaluating security policies, and synthesizing findings.
                </p>
              </div>
            </div>
          )}

          {error && (
            <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs flex items-start gap-3">
              <ShieldAlert className="w-5 h-5 flex-shrink-0 mt-0.5" />
              <div className="space-y-1">
                <div className="font-semibold">Review Request Failed</div>
                <div>{error}</div>
              </div>
            </div>
          )}

          {review && (
            <div className="space-y-6">
              {/* Verdict Card */}
              <div className="p-4 rounded-xl bg-forge-card border border-forge-border flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                  {review.verdict === 'APPROVE' && (
                    <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
                      <CheckCircle className="w-5 h-5" />
                    </div>
                  )}
                  {review.verdict === 'REQUEST_CHANGES' && (
                    <div className="w-10 h-10 rounded-xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-400">
                      <ShieldAlert className="w-5 h-5" />
                    </div>
                  )}
                  {review.verdict === 'COMMENT' && (
                    <div className="w-10 h-10 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-400">
                      <MessageSquare className="w-5 h-5" />
                    </div>
                  )}

                  <div>
                    <div className="text-xs text-forge-muted">Automated Verdict</div>
                    <div className="text-base font-bold text-white flex items-center gap-2">
                      <span>
                        {review.verdict === 'APPROVE' && 'Approve Changes'}
                        {review.verdict === 'REQUEST_CHANGES' && 'Changes Requested'}
                        {review.verdict === 'COMMENT' && 'Advisory Comments'}
                      </span>
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-4">
                  <div className="text-right">
                    <div className="text-[11px] text-forge-muted">Quality Score</div>
                    <div className="text-lg font-mono font-bold text-white">{review.score}/100</div>
                  </div>
                  <div className="w-20 bg-forge-bg h-2 rounded-full overflow-hidden border border-forge-border">
                    <div
                      className={`h-full rounded-full ${
                        review.score >= 80
                          ? 'bg-emerald-500'
                          : review.score >= 60
                          ? 'bg-amber-500'
                          : 'bg-rose-500'
                      }`}
                      style={{ width: `${review.score}%` }}
                    />
                  </div>
                </div>
              </div>

              {/* Summary */}
              <div className="space-y-2">
                <h4 className="text-xs font-semibold uppercase tracking-wider text-forge-muted">
                  Executive Assessment
                </h4>
                <p className="text-xs text-forge-text leading-relaxed bg-forge-bg/60 p-3 rounded-xl border border-forge-border">
                  {review.summary}
                </p>
              </div>

              {/* Findings Section */}
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-forge-muted">
                    Findings & Recommendations ({review.findings.length})
                  </h4>
                </div>

                {review.findings.length === 0 ? (
                  <div className="p-4 rounded-xl bg-emerald-500/5 border border-emerald-500/20 text-xs text-emerald-400 flex items-center gap-2">
                    <CheckCircle className="w-4 h-4" />
                    <span>No security vulnerabilities, bugs, or performance issues identified in this changeset.</span>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {review.findings.map((f, i) => (
                      <div
                        key={i}
                        className="p-3.5 rounded-xl bg-forge-card border border-forge-border space-y-2 text-xs"
                      >
                        <div className="flex items-center justify-between gap-2 flex-wrap">
                          <div className="flex items-center gap-2">
                            <span
                              className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${
                                f.severity === 'CRITICAL'
                                  ? 'bg-rose-500/20 text-rose-300 border border-rose-500/30'
                                  : f.severity === 'HIGH'
                                  ? 'bg-rose-500/15 text-rose-400 border border-rose-500/20'
                                  : f.severity === 'MEDIUM'
                                  ? 'bg-amber-500/15 text-amber-400 border border-amber-500/20'
                                  : 'bg-blue-500/15 text-blue-400 border border-blue-500/20'
                              }`}
                            >
                              {f.severity}
                            </span>
                            <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-forge-bg text-forge-muted border border-forge-border">
                              {f.category}
                            </span>
                            <span className="font-semibold text-white">{f.title}</span>
                          </div>

                          {f.file && (
                            <span className="text-[11px] font-mono text-forge-muted flex items-center gap-1">
                              <FileCode className="w-3 h-3 text-purple-400" />
                              {f.file}{f.line ? `:${f.line}` : ''}
                            </span>
                          )}
                        </div>

                        <p className="text-forge-text text-xs leading-relaxed">{f.description}</p>

                        {f.suggested_fix && (
                          <div className="mt-2 bg-forge-bg p-2.5 rounded-lg border border-forge-border/80 space-y-1">
                            <div className="text-[10px] uppercase font-semibold text-forge-muted">
                              Suggested Fix:
                            </div>
                            <div className="font-mono text-[11px] text-purple-300">
                              {f.suggested_fix}
                            </div>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Modal Footer */}
        {review && (
          <div className="p-4 border-t border-forge-border bg-forge-card/80 flex items-center justify-between">
            <button
              onClick={handleCopy}
              className="px-3 py-1.5 rounded-lg bg-forge-bg hover:bg-forge-border border border-forge-border text-xs text-forge-text flex items-center gap-1.5 transition-colors"
            >
              {copied ? (
                <>
                  <Check className="w-3.5 h-3.5 text-emerald-400" />
                  <span>Copied Markdown</span>
                </>
              ) : (
                <>
                  <Copy className="w-3.5 h-3.5" />
                  <span>Copy Markdown Review</span>
                </>
              )}
            </button>

            <div className="flex items-center space-x-2">
              <button
                onClick={() => runReview(false)}
                disabled={loading || posting}
                className="px-3 py-1.5 rounded-lg bg-forge-bg hover:bg-forge-border border border-forge-border text-xs text-forge-text transition-colors"
              >
                Re-run Review
              </button>

              <button
                onClick={handlePostReview}
                disabled={posting || postedSuccess}
                className="px-3.5 py-1.5 rounded-lg bg-purple-600 hover:bg-purple-500 disabled:opacity-50 text-white text-xs font-semibold shadow-md shadow-purple-600/20 transition-all flex items-center gap-1.5"
              >
                {posting ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    <span>Posting...</span>
                  </>
                ) : postedSuccess ? (
                  <>
                    <Check className="w-3.5 h-3.5 text-emerald-300" />
                    <span>Review Published!</span>
                  </>
                ) : (
                  <>
                    <Send className="w-3.5 h-3.5" />
                    <span>Publish Review to PR</span>
                  </>
                )}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
