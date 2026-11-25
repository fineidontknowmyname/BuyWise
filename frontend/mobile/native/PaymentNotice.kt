package com.buywise.app

import org.json.JSONObject
import java.math.BigDecimal
import java.time.Instant
import java.time.temporal.ChronoUnit

internal object PaymentNotice {
    private val excluded = Regex("\\b(?:otp|one[ -]time password|verification code|credited|received|refund|refunded|reversal|reversed|failed|declined|pending|request|requested|sent you|paid you|will be|scheduled|amount due)\\b", RegexOption.IGNORE_CASE)
    private val debit = Regex("\\b(?:paid|debited|spent|you sent)\\b", RegexOption.IGNORE_CASE)
    private val money = Regex("(?:₹|\\bINR\\s*|\\bRs\\.?\\s*)\\s*([0-9][0-9,]*(?:\\.[0-9]{1,2})?)(?![0-9.,])", RegexOption.IGNORE_CASE)
    private val reference = Regex("\\b(?:UTR|UPI ref(?:erence)?|transaction ID)\\s*[:#-]?\\s*([A-Z0-9]{8,40})\\b", RegexOption.IGNORE_CASE)
    private val formatted = Regex("(?:[0-9]+|[0-9]{1,3}(?:,[0-9]{3})+|[0-9]{1,2}(?:,[0-9]{2})*,[0-9]{3})(?:\\.[0-9]{1,2})?")

    fun parse(source: String, text: String, time: Instant): JSONObject? {
        if (text.length > 600 || excluded.containsMatchIn(text) || !debit.containsMatchIn(text)) return null
        val amounts = money.findAll(text).toList()
        val value = amounts.singleOrNull()?.groupValues?.get(1) ?: return null
        if (!formatted.matches(value)) return null
        val paise = runCatching { BigDecimal(value.replace(",", "")).movePointRight(2).longValueExact() }.getOrNull() ?: return null
        if (paise !in 1L..1_000_000_000_000L) return null
        return JSONObject().put("sourceApp", source).put("amountPaise", paise).put("currency", "INR")
            .put("occurredAt", time.truncatedTo(ChronoUnit.SECONDS).toString()).apply {
                reference.find(text)?.groupValues?.get(1)?.let { put("reference", it) }
            }
    }
}
