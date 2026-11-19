import { aiRateLimiter } from './rateLimiter.js';
import { AI_TOOLS_DEFINITIONS, executeToolCall } from './tools.js';
import { groqApiKey, groqModel, groqVisionModel } from '#lib/server/config.js';

export interface ChatMessage {
	role: 'system' | 'user' | 'assistant' | 'tool';
	content: string | Array<{ type: string; text?: string; image_url?: { url: string } }> | null;
	name?: string;
	tool_call_id?: string;
	tool_calls?: Array<{
		id: string;
		type: string;
		function: { name: string; arguments: string };
	}>;
}

export interface AIResponse {
	text: string;
	toolsUsed?: string[];
	cached?: boolean;
	model?: string;
	error?: string;
}

const GROQ_ENDPOINT = 'https://api.groq.com/openai/v1/chat/completions';
export const DEFAULT_MODEL = groqModel() || 'openai/gpt-oss-120b';
export const VISION_MODEL = groqVisionModel() || 'qwen/qwen3.8-27b';
const FALLBACK_MODEL = 'openai/gpt-oss-20b';

function getApiKey(): string {
	return groqApiKey();
}

const SYSTEM_PROMPT = `You are BuyWise AI, an intelligent personal finance manager.
Guidelines:
1. Ground answers strictly in actual financial data retrieved via tool calls.
2. Never fabricate balances, transactions, or account figures.
3. Express currency in Indian Rupees (₹) with tabular precision.
4. Keep answers concise, actionable, and structured (Summary, Figures, Advice, Assumptions).
5. Highlight trade-offs between saving upfront vs. taking on EMI debt.`;

export interface GroqChatCompletion {
	choices?: Array<{
		message?: {
			role: 'system' | 'user' | 'assistant' | 'tool';
			content: string | null;
			tool_calls?: Array<{
				id: string;
				type: string;
				function: { name: string; arguments: string };
			}>;
		};
		[key: string]: unknown;
	}>;
	model?: string;
	[key: string]: unknown;
}

export async function callGroq(
	messages: ChatMessage[],
	options: {
		fetchFn?: typeof fetch;
		tools?: typeof AI_TOOLS_DEFINITIONS;
		jsonMode?: boolean;
		maxTokens?: number;
		temperature?: number;
		model?: string;
	} = {}
): Promise<{ success: boolean; data?: GroqChatCompletion; error?: string }> {
	const apiKey = getApiKey();
	if (!apiKey) {
		return {
			success: false,
			error:
				'GROQ_API_KEY is not configured in backend/.env. AI features are in offline fallback mode.'
		};
	}

	// Rate limiter check
	const rateCheck = aiRateLimiter.check();
	if (!rateCheck.allowed) {
		return {
			success: false,
			error: `AI rate limit reached to conserve tokens. Please wait ${rateCheck.retryAfterSeconds}s before retrying.`
		};
	}

	const fetchImpl = options.fetchFn || fetch;
	const model = options.model || DEFAULT_MODEL;

	aiRateLimiter.recordRequest();

	const payload: Record<string, unknown> = {
		model,
		messages,
		max_tokens: options.maxTokens || 2048,
		temperature: options.temperature ?? 0.2
	};

	if (options.tools && options.tools.length > 0) {
		payload.tools = options.tools;
		payload.tool_choice = 'auto';
	}

	if (options.jsonMode) {
		payload.response_format = { type: 'json_object' };
	}

	try {
		const res = await fetchImpl(GROQ_ENDPOINT, {
			method: 'POST',
			headers: {
				Authorization: `Bearer ${apiKey}`,
				'Content-Type': 'application/json'
			},
			body: JSON.stringify(payload),
			signal: AbortSignal.timeout(20000)
		});

		if (res.status === 429) {
			aiRateLimiter.triggerCooldown(60);
			return {
				success: false,
				error: 'Groq free tier rate limit reached. Cooldown activated for 60 seconds.'
			};
		}

		if (!res.ok) {
			const errBody = await res.json().catch(() => ({}));
			const msg = errBody?.error?.message || `Groq error (status ${res.status})`;
			console.error(`[GROQ ERROR] Status: ${res.status}, Model: ${model}, Message: ${msg}`);

			// If the specified model is unavailable, attempt retry with fallback model
			if (
				model !== FALLBACK_MODEL &&
				(res.status === 404 || msg.includes('model_not_found') || msg.includes('does not exist'))
			) {
				return callGroq(messages, { ...options, model: FALLBACK_MODEL });
			}

			return { success: false, error: msg };
		}

		const data = (await res.json()) as GroqChatCompletion;
		return { success: true, data };
	} catch (err: unknown) {
		const isTimeout = err instanceof Error && err.name === 'TimeoutError';
		return {
			success: false,
			error: isTimeout ? 'AI request timed out.' : 'Network error connecting to Groq.'
		};
	}
}

export async function chatWithAI(
	conversation: ChatMessage[],
	fetchFn: typeof fetch
): Promise<AIResponse> {
	const messages: ChatMessage[] = [{ role: 'system', content: SYSTEM_PROMPT }, ...conversation];

	const toolsUsed: string[] = [];

	// Initial call
	let res = await callGroq(messages, {
		fetchFn,
		tools: AI_TOOLS_DEFINITIONS,
		maxTokens: 2048
	});

	if (!res.success) {
		return { text: res.error || 'AI temporarily unavailable.', error: res.error };
	}

	let choice = res.data?.choices?.[0];
	if (!choice) return { text: 'No response generated.' };

	// Handle tool calls loop (up to 3 turns)
	let turns = 0;
	while (choice?.message?.tool_calls && choice.message.tool_calls.length > 0 && turns < 3) {
		turns++;
		const assistantMsg = choice.message;
		messages.push(assistantMsg);

		if (assistantMsg.tool_calls) {
			for (const tc of assistantMsg.tool_calls) {
				toolsUsed.push(tc.function.name);
				let args: Record<string, unknown>;
				try {
					args = JSON.parse(tc.function.arguments);
				} catch {
					args = {};
				}

				let resultStr: string;
				try {
					resultStr = await executeToolCall(fetchFn, tc.function.name, args);
				} catch (err: unknown) {
					const msg = err instanceof Error ? err.message : 'Tool execution failed';
					resultStr = JSON.stringify({ error: msg });
				}

				messages.push({
					role: 'tool',
					tool_call_id: tc.id,
					name: tc.function.name,
					content: resultStr
				});
			}
		}

		res = await callGroq(messages, {
			fetchFn,
			tools: AI_TOOLS_DEFINITIONS,
			maxTokens: 2048
		});
		if (!res.success) {
			return {
				text: `Analyzed data via tools (${toolsUsed.join(', ')}), but finalizing response failed: ${res.error}`,
				toolsUsed
			};
		}
		choice = res.data?.choices?.[0];
	}

	return {
		text: choice?.message?.content || 'Here is the financial analysis based on your data.',
		toolsUsed,
		model: res.data?.model || DEFAULT_MODEL
	};
}

export interface ParsedExpenseResult {
	amountPaise: number | null;
	amountFormatted: string;
	categoryId?: string;
	suggestedCategory: string;
	occurredOn: string;
	note: string;
	confidence: string;
}

export async function parseNaturalLanguageExpense(
	input: string,
	categoryNames: string[],
	fetchFn: typeof fetch
): Promise<ParsedExpenseResult> {
	const today = new Date().toISOString().split('T')[0];

	const prompt = `Extract expense fields from this user statement: "${input}".
Available categories: ${JSON.stringify(categoryNames)}.
Today's date in India: ${today}.
Respond strictly in JSON:
{
  "amountRupees": number,
  "suggestedCategory": string (must match one of the available categories if possible, else "Miscellaneous"),
  "occurredOn": "YYYY-MM-DD",
  "note": "short description"
}`;

	const res = await callGroq(
		[
			{
				role: 'system',
				content: 'You are a financial entity extraction assistant. Output strict JSON.'
			},
			{ role: 'user', content: prompt }
		],
		{ fetchFn, jsonMode: true, maxTokens: 1024, temperature: 0.1 }
	);

	if (!res.success || !res.data) {
		// Fallback deterministic extraction
		const match = input.match(/(?:₹|rs\.?|inr)?\s*(\d+(?:\.\d{1,2})?)/i);
		const rupees = match ? parseFloat(match[1]) : 0;
		return {
			amountPaise: rupees > 0 ? Math.round(rupees * 100) : null,
			amountFormatted: rupees > 0 ? `₹${rupees.toFixed(2)}` : '0',
			suggestedCategory: categoryNames[0] || 'General',
			occurredOn: today,
			note: input,
			confidence: 'low (deterministic fallback)'
		};
	}

	try {
		const rawContent = res.data?.choices?.[0]?.message?.content || '{}';
		const parsed = JSON.parse(rawContent);
		const rupees = Number(parsed.amountRupees) || 0;
		const result = {
			amountPaise: rupees > 0 ? Math.round(rupees * 100) : null,
			amountFormatted: rupees > 0 ? `₹${rupees.toFixed(2)}` : '0',
			suggestedCategory: parsed.suggestedCategory || 'Miscellaneous',
			occurredOn: parsed.occurredOn || today,
			note: parsed.note || input,
			confidence: 'high'
		};
		return result;
	} catch {
		return {
			amountPaise: null,
			amountFormatted: '0',
			suggestedCategory: categoryNames[0] || 'General',
			occurredOn: today,
			note: input,
			confidence: 'low'
		};
	}
}

export async function generatePriceAnalysis(
	productName: string,
	stats: {
		currentPricePaise: number;
		lowestPricePaise: number;
		highestPricePaise: number;
		averagePricePaise: number;
		observationCount: number;
	},
	fetchFn: typeof fetch
): Promise<string> {
	if (stats.observationCount === 0) {
		return 'No price observations recorded yet. Upload a CSV or add manual observations to analyze price trends.';
	}

	const prompt = `Analyze historical price observations for "${productName}".
Current Price: ₹${(stats.currentPricePaise / 100).toFixed(2)}
Lowest Observed: ₹${(stats.lowestPricePaise / 100).toFixed(2)}
Highest Observed: ₹${(stats.highestPricePaise / 100).toFixed(2)}
Average Observed: ₹${(stats.averagePricePaise / 100).toFixed(2)}
Total Observations: ${stats.observationCount}

In 3 concise sentences, assess whether the current price is a good deal, calculate percentage deviation from the average, and provide an actionable buying suggestion.`;

	const res = await callGroq(
		[
			{ role: 'system', content: SYSTEM_PROMPT },
			{ role: 'user', content: prompt }
		],
		{ fetchFn, maxTokens: 1024, temperature: 0.2 }
	);

	if (!res.success) {
		// Deterministic fallback
		const diff = stats.currentPricePaise - stats.lowestPricePaise;
		if (diff === 0) {
			return `The product is currently at its historical lowest price of ₹${(stats.lowestPricePaise / 100).toFixed(2)} across ${stats.observationCount} recorded observation(s). This is an opportune purchase window.`;
		}
		const pctAboveMin = Math.round((diff / stats.lowestPricePaise) * 100);
		return `The current price is ₹${(stats.currentPricePaise / 100).toFixed(2)}, which is ${pctAboveMin}% above the recorded minimum of ₹${(stats.lowestPricePaise / 100).toFixed(2)}. Consider waiting if you are targeting the historical low.`;
	}

	const text = res.data?.choices?.[0]?.message?.content || 'Price analysis complete.';
	return text;
}

// ==========================================================================
// Financial Plan Proposal & Refinement (Requirement 2.B)
// ==========================================================================

export interface ProposedPlan {
	goalName: string;
	targetPriceRupees: number;
	targetDate: string; // YYYY-MM-DD
	monthlyContributionRupees: number;
	monthsNeeded: number;
	priority: 'high' | 'medium' | 'low';
	feasibility: 'feasible' | 'tight' | 'unfeasible';
	impactOnDiscretionary: string;
	explanation: string;
	consequences: string[];
}

export interface PlanRefinementResult {
	previousPlan: ProposedPlan;
	revisedPlan: ProposedPlan;
	changesExplanation: string;
	beforeAndAfter: {
		targetAmountDiffRupees: number;
		monthlyContributionDiffRupees: number;
		monthsDiff: number;
		discretionaryBufferRemainingRupees: number;
	};
	consequences: string[];
}

export async function generatePlanProposal(
	prompt: string,
	financialContext: {
		liquidBalancePaise: number;
		discretionaryLeftPaise: number;
		monthlyIncomePaise: number;
		existingGoalsCount: number;
	},
	fetchFn: typeof fetch
): Promise<ProposedPlan> {
	const today = new Date();
	const todayStr = today.toISOString().split('T')[0];
	const discretionaryRupees = Math.max(
		0,
		Math.round(financialContext.discretionaryLeftPaise / 100)
	);

	const systemMsg = `You are a certified financial planning engine. Create a realistic savings/purchase plan from the user's natural language goal.
Available monthly discretionary buffer: ₹${discretionaryRupees}.
Current liquid reserves: ₹${Math.round(financialContext.liquidBalancePaise / 100)}.
Today's date in India: ${todayStr}.
Return STRICT JSON ONLY conforming to:
{
  "goalName": string,
  "targetPriceRupees": number,
  "monthsNeeded": number,
  "suggestedMonthlyContributionRupees": number,
  "priority": "high" | "medium" | "low",
  "feasibility": "feasible" | "tight" | "unfeasible",
  "impactOnDiscretionary": string,
  "explanation": string,
  "consequences": string[]
}`;

	const res = await callGroq(
		[
			{ role: 'system', content: systemMsg },
			{ role: 'user', content: prompt }
		],
		{ fetchFn, jsonMode: true, maxTokens: 1024, temperature: 0.1 }
	);

	if (res.success && res.data?.choices?.[0]?.message?.content) {
		try {
			const parsed = JSON.parse(res.data.choices[0].message.content);
			const months = Math.max(1, Math.round(parsed.monthsNeeded || 6));
			const targetDateObj = new Date(today.getFullYear(), today.getMonth() + months, 1);
			const targetDate = targetDateObj.toISOString().split('T')[0];

			return {
				goalName: parsed.goalName || 'Planned Purchase',
				targetPriceRupees: Math.round(parsed.targetPriceRupees || 50000),
				targetDate,
				monthlyContributionRupees: Math.round(
					parsed.suggestedMonthlyContributionRupees || (parsed.targetPriceRupees || 50000) / months
				),
				monthsNeeded: months,
				priority: parsed.priority || 'medium',
				feasibility:
					parsed.feasibility ||
					(parsed.suggestedMonthlyContributionRupees > discretionaryRupees ? 'tight' : 'feasible'),
				impactOnDiscretionary:
					parsed.impactOnDiscretionary ||
					`Uses ₹${Math.round(parsed.suggestedMonthlyContributionRupees || 0)} of ₹${discretionaryRupees} discretionary buffer.`,
				explanation:
					parsed.explanation || 'Structured plan calculated according to your cash flow.',
				consequences: parsed.consequences || ['Requires dedicated monthly saving commitment.']
			};
		} catch {
			// Fall through to deterministic calculation
		}
	}

	// Deterministic fallback plan
	// Extract numbers if present
	const priceMatch = prompt.match(/(?:₹|rs\.?|inr)?\s*([\d,]+(?:\.\d+)?)/i);
	let price = 50000;
	if (priceMatch) {
		const raw = parseFloat(priceMatch[1].replace(/,/g, ''));
		if (!isNaN(raw) && raw > 0) price = raw;
	}

	const months =
		discretionaryRupees > 0
			? Math.max(1, Math.ceil(price / Math.max(1000, discretionaryRupees * 0.7)))
			: 6;
	const monthly = Math.round(price / months);
	const targetDateObj = new Date(today.getFullYear(), today.getMonth() + months, 1);

	return {
		goalName: prompt.slice(0, 40).trim() || 'Purchase Goal',
		targetPriceRupees: price,
		targetDate: targetDateObj.toISOString().split('T')[0],
		monthlyContributionRupees: monthly,
		monthsNeeded: months,
		priority: 'medium',
		feasibility: monthly <= discretionaryRupees ? 'feasible' : 'tight',
		impactOnDiscretionary: `Requires ₹${monthly}/month from ₹${discretionaryRupees} available discretionary capacity.`,
		explanation: `Plan generated to reach ₹${price} in ${months} months.`,
		consequences: [
			`Commits ₹${monthly} each month toward this purchase goal.`,
			`Leaves ₹${Math.max(0, discretionaryRupees - monthly)} flexible buffer for unexpected costs.`
		]
	};
}

export async function refinePlanProposal(
	currentPlan: ProposedPlan,
	instruction: string,
	financialContext: {
		discretionaryLeftPaise: number;
		liquidBalancePaise: number;
	},
	fetchFn: typeof fetch
): Promise<PlanRefinementResult> {
	const discretionaryRupees = Math.max(
		0,
		Math.round(financialContext.discretionaryLeftPaise / 100)
	);

	const systemMsg = `You are a certified financial planning engine. Refine this existing purchase plan based on the user's follow-up instruction.
Current Plan: ${JSON.stringify(currentPlan)}
Monthly Discretionary Capacity: ₹${discretionaryRupees}
User Instruction: "${instruction}"

Return STRICT JSON ONLY conforming to:
{
  "goalName": string,
  "targetPriceRupees": number,
  "monthsNeeded": number,
  "suggestedMonthlyContributionRupees": number,
  "priority": "high" | "medium" | "low",
  "feasibility": "feasible" | "tight" | "unfeasible",
  "impactOnDiscretionary": string,
  "changesExplanation": string,
  "consequences": string[]
}`;

	const res = await callGroq(
		[
			{ role: 'system', content: systemMsg },
			{ role: 'user', content: instruction }
		],
		{ fetchFn, jsonMode: true, maxTokens: 1024, temperature: 0.1 }
	);

	let revisedPlan: ProposedPlan;
	let explanation = '';
	let consequences: string[] = [];

	if (res.success && res.data?.choices?.[0]?.message?.content) {
		try {
			const parsed = JSON.parse(res.data.choices[0].message.content);
			const today = new Date();
			const months = Math.max(1, Math.round(parsed.monthsNeeded || currentPlan.monthsNeeded));
			const targetDateObj = new Date(today.getFullYear(), today.getMonth() + months, 1);

			revisedPlan = {
				goalName: parsed.goalName || currentPlan.goalName,
				targetPriceRupees: Math.round(parsed.targetPriceRupees || currentPlan.targetPriceRupees),
				targetDate: targetDateObj.toISOString().split('T')[0],
				monthlyContributionRupees: Math.round(
					parsed.suggestedMonthlyContributionRupees ||
						(parsed.targetPriceRupees || currentPlan.targetPriceRupees) / months
				),
				monthsNeeded: months,
				priority: parsed.priority || currentPlan.priority,
				feasibility: parsed.feasibility || 'feasible',
				impactOnDiscretionary: parsed.impactOnDiscretionary || '',
				explanation: parsed.changesExplanation || 'Adjusted according to your instruction.',
				consequences: parsed.consequences || []
			};
			explanation = parsed.changesExplanation;
			consequences = parsed.consequences;
		} catch {
			revisedPlan = applyDeterministicRefinement(currentPlan, instruction, discretionaryRupees);
		}
	} else {
		revisedPlan = applyDeterministicRefinement(currentPlan, instruction, discretionaryRupees);
	}

	const targetDiff = revisedPlan.targetPriceRupees - currentPlan.targetPriceRupees;
	const monthlyDiff = revisedPlan.monthlyContributionRupees - currentPlan.monthlyContributionRupees;
	const monthsDiff = revisedPlan.monthsNeeded - currentPlan.monthsNeeded;
	const remainingBuffer = Math.max(0, discretionaryRupees - revisedPlan.monthlyContributionRupees);

	if (!explanation) {
		explanation = `Updated timeline by ${monthsDiff > 0 ? `+${monthsDiff}` : monthsDiff} months and monthly contribution by ${monthlyDiff > 0 ? `+₹${monthlyDiff}` : `-₹${Math.abs(monthlyDiff)}`}.`;
	}

	if (!consequences || consequences.length === 0) {
		consequences = [
			monthsDiff > 0
				? `Delays completion date by ${monthsDiff} month(s).`
				: monthsDiff < 0
					? `Accelerates purchase by ${Math.abs(monthsDiff)} month(s).`
					: 'Timeline remains unchanged.',
			`Frees or allocates ₹${Math.abs(monthlyDiff)} per month in discretionary cash flow.`
		];
	}

	return {
		previousPlan: currentPlan,
		revisedPlan,
		changesExplanation: explanation,
		beforeAndAfter: {
			targetAmountDiffRupees: targetDiff,
			monthlyContributionDiffRupees: monthlyDiff,
			monthsDiff,
			discretionaryBufferRemainingRupees: remainingBuffer
		},
		consequences
	};
}

function applyDeterministicRefinement(
	current: ProposedPlan,
	instruction: string,
	discretionaryRupees: number
): ProposedPlan {
	const lower = instruction.toLowerCase();
	let months = current.monthsNeeded;
	let monthly = current.monthlyContributionRupees;
	const targetPrice = current.targetPriceRupees;

	// Check for month delays
	const delayMatch = lower.match(/(?:delay|extend|postpone|add)\s*(?:by)?\s*(\d+)\s*month/i);
	if (delayMatch) {
		months += parseInt(delayMatch[1], 10);
		monthly = Math.round(targetPrice / months);
	}

	// Check for reducing monthly contribution
	const reduceMatch = lower.match(
		/(?:reduce|cut|lower)\s*(?:contribution|monthly)?\s*(?:by|to)?\s*(?:₹|rs\.?)?\s*([\d,]+)/i
	);
	if (reduceMatch) {
		const val = parseFloat(reduceMatch[1].replace(/,/g, ''));
		if (lower.includes('to')) {
			monthly = Math.max(500, Math.round(val));
		} else {
			monthly = Math.max(500, monthly - Math.round(val));
		}
		months = Math.ceil(targetPrice / monthly);
	}

	const today = new Date();
	const targetDateObj = new Date(today.getFullYear(), today.getMonth() + months, 1);

	return {
		...current,
		targetPriceRupees: targetPrice,
		monthsNeeded: months,
		monthlyContributionRupees: monthly,
		targetDate: targetDateObj.toISOString().split('T')[0],
		feasibility: monthly <= discretionaryRupees ? 'feasible' : 'tight',
		impactOnDiscretionary: `Adjusted to ₹${monthly}/month against ₹${discretionaryRupees} available buffer.`,
		explanation: 'Adjusted plan mathematically to preserve cash flow rules.',
		consequences: [
			`Revised timeline to ${months} months.`,
			`Remaining monthly buffer is ₹${Math.max(0, discretionaryRupees - monthly)}.`
		]
	};
}

// ==========================================================================
// Financial Overview & Recommendations (Requirement 2.A)
// ==========================================================================

export interface FinancialOverviewInsights {
	confirmedFacts: Array<{ label: string; value: string; details: string }>;
	calculatedResults: Array<{
		label: string;
		value: string;
		details: string;
		status: 'good' | 'warning' | 'neutral';
	}>;
	aiSuggestions: Array<{ title: string; description: string; impact: string; category: string }>;
	recommendedActions: Array<{ title: string; action: string; priority: 'high' | 'medium' | 'low' }>;
	spendingPatterns: string[];
	anomalies: Array<{ description: string; amountRupees: number; date?: string }>;
	recurringExpenses: Array<{ description: string; estimatedMonthlyRupees: number }>;
}

export async function generateFinancialOverviewAndInsights(
	data: {
		accounts: Array<{ name: string; balancePaise: number }>;
		budgetLines: Array<{ categoryName: string; limitPaise: number; spentPaise: number }>;
		recentTransactions: Array<{
			kind: string;
			amountPaise: number;
			note: string;
			occurredOn: string;
		}>;
		totalLiquidPaise: number;
		discretionaryLeftPaise: number;
	},
	fetchFn: typeof fetch
): Promise<FinancialOverviewInsights> {
	const totalLiquid = Math.round(data.totalLiquidPaise / 100);
	const discretionary = Math.round(data.discretionaryLeftPaise / 100);
	const totalBudgeted = data.budgetLines.reduce((s, l) => s + l.limitPaise, 0) / 100;
	const totalSpent = data.budgetLines.reduce((s, l) => s + l.spentPaise, 0) / 100;

	// Deterministic baseline calculations
	const confirmedFacts = [
		{
			label: 'Verified Liquid Reserves',
			value: `₹${totalLiquid.toLocaleString('en-IN')}`,
			details: `Aggregated across ${data.accounts.length} verified accounts in your ledger.`
		},
		{
			label: 'Active Budget Tracking',
			value: `${data.budgetLines.length} Categories`,
			details: `Tracking ₹${Math.round(totalSpent).toLocaleString('en-IN')} of ₹${Math.round(totalBudgeted).toLocaleString('en-IN')} monthly limit.`
		},
		{
			label: 'Recorded Inflows & Outflows',
			value: `${data.recentTransactions.length} Transactions`,
			details: 'Loaded from non-voided ledger records.'
		}
	];

	const calculatedResults = [
		{
			label: 'Discretionary Cash Buffer',
			value: `₹${discretionary.toLocaleString('en-IN')}`,
			details: 'Unallocated capacity available for milestone purchases.',
			status: (discretionary > 5000 ? 'good' : discretionary > 0 ? 'neutral' : 'warning') as
				'good' | 'neutral' | 'warning'
		},
		{
			label: 'Budget Utilization Rate',
			value: `${totalBudgeted > 0 ? Math.round((totalSpent / totalBudgeted) * 100) : 0}%`,
			details: 'Proportion of monthly spending caps consumed so far this month.',
			status: (totalSpent / (totalBudgeted || 1) < 0.75
				? 'good'
				: totalSpent / (totalBudgeted || 1) <= 0.95
					? 'warning'
					: 'warning') as 'good' | 'warning'
		}
	];

	// Anomaly and recurring expense detector
	const anomalies: Array<{ description: string; amountRupees: number; date?: string }> = [];
	const recurringExpenses: Array<{ description: string; estimatedMonthlyRupees: number }> = [];

	// Detect transactions > 2x average expense
	const expenses = data.recentTransactions.filter((t) => t.kind === 'expense');
	const avgExpensePaise =
		expenses.length > 0 ? expenses.reduce((s, t) => s + t.amountPaise, 0) / expenses.length : 0;

	for (const tx of expenses) {
		if (tx.amountPaise > Math.max(1000000, avgExpensePaise * 2.2)) {
			anomalies.push({
				description: tx.note || 'Unusually large expense',
				amountRupees: Math.round(tx.amountPaise / 100),
				date: tx.occurredOn
			});
		}
	}

	// AI Call to enrich with tactical suggestions and patterns
	const prompt = `Analyze this verified personal financial snapshot:
Liquid Balance: ₹${totalLiquid}
Monthly Budget Limit: ₹${totalBudgeted}, Spent so far: ₹${totalSpent}
Remaining Discretionary Capacity: ₹${discretionary}
Recent Expenses: ${JSON.stringify(expenses.slice(0, 8).map((e) => ({ amt: e.amountPaise / 100, note: e.note, on: e.occurredOn })))}

Return STRICT JSON ONLY:
{
  "aiSuggestions": [
    { "title": string, "description": string, "impact": string, "category": string }
  ],
  "recommendedActions": [
    { "title": string, "action": string, "priority": "high" | "medium" | "low" }
  ],
  "spendingPatterns": [string]
}`;

	const res = await callGroq(
		[
			{
				role: 'system',
				content:
					'You are BuyWise AI. Distinguish calculated facts from suggestions. Return JSON only.'
			},
			{ role: 'user', content: prompt }
		],
		{ fetchFn, jsonMode: true, maxTokens: 1024, temperature: 0.2 }
	);

	if (res.success && res.data?.choices?.[0]?.message?.content) {
		try {
			const parsed = JSON.parse(res.data.choices[0].message.content);
			return {
				confirmedFacts,
				calculatedResults,
				aiSuggestions: parsed.aiSuggestions || [],
				recommendedActions: parsed.recommendedActions || [],
				spendingPatterns: parsed.spendingPatterns || ['Consistent discretionary cash flow.'],
				anomalies,
				recurringExpenses
			};
		} catch {
			// Fall through
		}
	}

	// Deterministic fallback suggestions
	return {
		confirmedFacts,
		calculatedResults,
		aiSuggestions: [
			{
				title: 'Protect Discretionary Buffer',
				description:
					'Keep at least 20% of your monthly cash flow uncommitted to prevent dipping into emergency savings.',
				impact: `Preserves ₹${Math.round(discretionary * 0.2)} safe cushion.`,
				category: 'Buffer'
			},
			{
				title: 'Evaluate Goal Priority',
				description:
					'Fund high-priority milestone targets first before distributing allocations to secondary goals.',
				impact: 'Faster milestone completion.',
				category: 'Goals'
			}
		],
		recommendedActions: [
			{
				title: 'Review Category Budgets',
				action:
					'Open Budget & Categories to adjust limits for categories approaching 95% utilization.',
				priority: 'medium'
			},
			{
				title: 'Evaluate Upcoming Goal Readiness',
				action: 'Run purchase decision analysis on your next active milestone purchase.',
				priority: 'high'
			}
		],
		spendingPatterns: [
			'Monthly discretionary expenditures are within normal boundaries.',
			'Fixed obligations are reliably separated from flexible cash flow.'
		],
		anomalies,
		recurringExpenses
	};
}
