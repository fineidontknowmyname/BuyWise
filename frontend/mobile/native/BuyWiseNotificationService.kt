package com.buywise.app

import android.app.Notification
import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import org.json.JSONArray
import java.time.Instant
import java.util.Locale

class BuyWiseNotificationService : NotificationListenerService() {
    override fun onNotificationPosted(notification: StatusBarNotification) {
        if (notification.notification.flags and Notification.FLAG_GROUP_SUMMARY != 0) return
        runCatching {
            val state = Vault.read(this)
            val session = state.optString("sessionId")
            if (session.isEmpty() || !state.optBoolean("captureEnabled") || !isAllowed(state.optJSONArray("packages"), notification.packageName)) return
            val extras = notification.notification.extras
            val text = (extras.getCharSequence(Notification.EXTRA_BIG_TEXT) ?: extras.getCharSequence(Notification.EXTRA_TEXT))?.toString() ?: return
            val event = PaymentNotice.parse(notification.packageName, text, Instant.ofEpochMilli(notification.postTime)) ?: return
            val reference = event.optString("reference").uppercase(Locale.ROOT)
            Vault.update(this) { latest ->
                if (latest.optString("sessionId") == session && latest.optBoolean("captureEnabled") && isAllowed(latest.optJSONArray("packages"), notification.packageName)) {
                    val queue = latest.optJSONArray("queue") ?: JSONArray()
                    val exists = (0 until queue.length()).any { index ->
                        val previous = queue.getJSONObject(index)
                        if (reference.isNotEmpty()) previous.optString("sourceApp") == notification.packageName && previous.optString("reference").uppercase(Locale.ROOT) == reference
                        else previous.toString() == event.toString()
                    }
                    if (!exists && queue.length() < 100) latest.put("queue", queue.put(event))
                }
            }
        }
    }

    private fun isAllowed(list: JSONArray?, app: String) = list != null && (0 until list.length()).any { list.optString(it) == app }
}
