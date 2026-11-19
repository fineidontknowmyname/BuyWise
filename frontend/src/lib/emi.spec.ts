import { describe, expect, it } from 'vitest';
import { calculateEMI } from './emi';

describe('calculateEMI', () => {
	it('handles zero interest loans correctly', () => {
		const result = calculateEMI({
			productPricePaise: 6000000, // ₹60,000
			downPaymentPaise: 0,
			annualRatePercent: 0,
			tenureMonths: 6
		});
		expect(result.monthlyInstallmentPaise).toBe(1000000); // ₹10,000/mo
		expect(result.totalInterestPaise).toBe(0);
		expect(result.totalRepaymentPaise).toBe(6000000);
		expect(result.extraCostVsCashPaise).toBe(0);
	});

	it('computes standard compound interest EMI accurately', () => {
		// ₹1,20,000 at 12% for 12 months
		const result = calculateEMI({
			productPricePaise: 12000000,
			downPaymentPaise: 2000000, // ₹20,000 down payment -> ₹1,00,000 principal
			annualRatePercent: 12,
			tenureMonths: 12,
			processingFeePaise: 100000 // ₹1,000 fee
		});

		expect(result.principalPaise).toBe(10000000);
		// Approx ₹8,885 per month
		expect(result.monthlyInstallmentPaise).toBeGreaterThan(880000);
		expect(result.monthlyInstallmentPaise).toBeLessThan(890000);
		expect(result.totalInterestPaise).toBeGreaterThan(600000);
		expect(result.processingFeePaise).toBe(100000);
		expect(result.upfrontOutflowPaise).toBe(2100000); // 20k down + 1k fee
	});

	it('evaluates scenarios comparison correctly', () => {
		const result = calculateEMI({
			productPricePaise: 5000000,
			downPaymentPaise: 1000000,
			annualRatePercent: 14,
			tenureMonths: 12
		});

		expect(result.scenarios.cash.totalCostPaise).toBe(5000000);
		expect(result.scenarios.savings.totalCostPaise).toBe(5000000);
		expect(result.scenarios.emi.totalCostPaise).toBeGreaterThan(5000000);
		expect(result.scenarios.emi.extraCostPaise).toBeGreaterThan(0);
	});
});
