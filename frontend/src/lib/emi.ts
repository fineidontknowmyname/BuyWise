// Deterministic EMI & Purchase Affordability Simulator

export interface EMISimulationInput {
	productPricePaise: number;
	downPaymentPaise: number;
	annualRatePercent: number; // e.g. 13.5
	tenureMonths: number; // e.g. 6, 12, 18, 24
	processingFeePaise?: number;
	monthlyDiscretionaryBudgetPaise?: number;
}

export interface EMISimulationResult {
	principalPaise: number;
	monthlyInstallmentPaise: number;
	totalInterestPaise: number;
	processingFeePaise: number;
	totalRepaymentPaise: number;
	upfrontOutflowPaise: number;
	extraCostVsCashPaise: number;
	budgetUtilizationPercent: number;
	monthsToSaveUpfront: number;
	scenarios: {
		cash: {
			upfrontPaise: number;
			monthlyPaise: number;
			totalCostPaise: number;
			extraCostPaise: number;
		};
		savings: {
			upfrontPaise: number;
			monthlyPaise: number;
			monthsRequired: number;
			totalCostPaise: number;
			extraCostPaise: number;
		};
		emi: {
			upfrontPaise: number;
			monthlyPaise: number;
			monthsRequired: number;
			totalCostPaise: number;
			extraCostPaise: number;
		};
	};
}

export function calculateEMI(input: EMISimulationInput): EMISimulationResult {
	const price = Math.max(0, input.productPricePaise);
	const downPayment = Math.min(price, Math.max(0, input.downPaymentPaise));
	const principal = Math.max(0, price - downPayment);
	const fee = Math.max(0, input.processingFeePaise || 0);
	const months = Math.max(1, Math.round(input.tenureMonths));
	const annualRate = Math.max(0, input.annualRatePercent);
	const budget = input.monthlyDiscretionaryBudgetPaise || 0;

	let monthlyPaise: number;
	let totalInterestPaise: number;

	if (principal === 0) {
		monthlyPaise = 0;
		totalInterestPaise = 0;
	} else if (annualRate === 0) {
		monthlyPaise = Math.round(principal / months);
		totalInterestPaise = 0;
	} else {
		const r = annualRate / (12 * 100);
		const factor = Math.pow(1 + r, months);
		const exactMonthly = (principal * r * factor) / (factor - 1);
		monthlyPaise = Math.round(exactMonthly);
		totalInterestPaise = Math.max(0, monthlyPaise * months - principal);
	}

	const totalRepaymentPaise = downPayment + principal + totalInterestPaise + fee;
	const upfrontOutflowPaise = downPayment + fee;
	const extraCostVsCashPaise = totalRepaymentPaise - price;

	const budgetUtilizationPercent =
		budget > 0 ? Math.min(100, Math.round((monthlyPaise / budget) * 100)) : 0;

	// Savings scenario: how many months if saving the same monthly installment
	const monthlySavingsRate = monthlyPaise > 0 ? monthlyPaise : Math.max(1, Math.round(price / 12));
	const monthsToSaveUpfront = Math.ceil(price / monthlySavingsRate);

	return {
		principalPaise: principal,
		monthlyInstallmentPaise: monthlyPaise,
		totalInterestPaise,
		processingFeePaise: fee,
		totalRepaymentPaise,
		upfrontOutflowPaise,
		extraCostVsCashPaise,
		budgetUtilizationPercent,
		monthsToSaveUpfront,
		scenarios: {
			cash: {
				upfrontPaise: price,
				monthlyPaise: 0,
				totalCostPaise: price,
				extraCostPaise: 0
			},
			savings: {
				upfrontPaise: 0,
				monthlyPaise: monthlySavingsRate,
				monthsRequired: monthsToSaveUpfront,
				totalCostPaise: price,
				extraCostPaise: 0
			},
			emi: {
				upfrontPaise: upfrontOutflowPaise,
				monthlyPaise,
				monthsRequired: months,
				totalCostPaise: totalRepaymentPaise,
				extraCostPaise: extraCostVsCashPaise
			}
		}
	};
}

export interface AmortizationRow {
	month: number;
	openingBalancePaise: number;
	emiPaise: number;
	principalPaise: number;
	interestPaise: number;
	closingBalancePaise: number;
}

export function generateAmortizationSchedule(
	principalPaise: number,
	annualRatePercent: number,
	tenureMonths: number,
	monthlyInstallmentPaise: number
): AmortizationRow[] {
	if (principalPaise <= 0 || tenureMonths <= 0) return [];

	const monthlyRate = annualRatePercent / (12 * 100);
	let balance = principalPaise;
	const schedule: AmortizationRow[] = [];

	for (let m = 1; m <= tenureMonths; m++) {
		const opening = balance;
		const interest = Math.round(opening * monthlyRate);
		let principal = monthlyInstallmentPaise - interest;

		if (m === tenureMonths || opening + interest <= monthlyInstallmentPaise) {
			principal = opening;
			balance = 0;
		} else {
			balance = Math.max(0, opening - principal);
		}

		schedule.push({
			month: m,
			openingBalancePaise: opening,
			emiPaise: principal + interest,
			principalPaise: principal,
			interestPaise: interest,
			closingBalancePaise: balance
		});

		if (balance === 0) break;
	}

	return schedule;
}
