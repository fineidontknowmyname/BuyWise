package com.buywise.app

import android.content.ComponentName
import android.content.Intent
import android.net.Uri
import android.provider.Settings
import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin
import org.json.JSONArray
import org.json.JSONObject
import org.json.JSONTokener
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URI
import java.time.Instant
import java.util.UUID
import java.util.concurrent.Executors

@CapacitorPlugin(name = "BuyWise")
class BuyWisePlugin : Plugin() {
    private val worker = Executors.newSingleThreadExecutor()
    private val uuid = "[a-fA-F0-9-]{36}"
    private val readPaths = Regex("(?:accounts|categories|goals|financial-profile|budgets|expenses/candidates|goals/$uuid/(?:decision|plan))")
    private val writePaths = Regex("(?:expenses|expenses/candidates/$uuid/(?:confirm|reject))")
    private val packageId = Regex("[A-Za-z][A-Za-z0-9_]*(?:\\.[A-Za-z][A-Za-z0-9_]*)+")

    private data class Result(val status: Int, val text: String) { val ok get() = status in 200..299 }
    private class Problem(message: String) : RuntimeException(message)

    private fun run(call: PluginCall, action: () -> JSObject) {
        worker.execute {
            try { call.resolve(action()) }
            catch (problem: Problem) { call.reject(problem.message ?: "BuyWise could not complete this request.") }
            catch (_: IOException) { call.reject("Unable to connect to BuyWise. Check your connection and try again.") }
            catch (_: Exception) { call.reject("BuyWise could not read or save its mobile data.") }
        }
    }

    private fun server(): URI {
        val uri = runCatching { URI(BuildConfig.BUYWISE_SERVER_URL) }.getOrNull()
        if (uri == null || uri.host.isNullOrBlank() || uri.rawUserInfo != null || uri.rawQuery != null || uri.rawFragment != null ||
            uri.path !in listOf("", "/") || (uri.scheme != "https" && !(BuildConfig.DEBUG && uri.scheme == "http" && uri.host == "10.0.2.2")))
            throw Problem("This Android build needs a valid BuyWise HTTPS server URL.")
        return uri
    }

    private fun http(path: String, method: String, token: String? = null, body: JSONObject? = null, key: String? = null): Result {
        val connection = server().resolve("/api/mobile/$path").toURL().openConnection() as HttpURLConnection
        connection.instanceFollowRedirects = false
        connection.connectTimeout = 7000
        connection.readTimeout = 10000
        connection.requestMethod = method
        connection.setRequestProperty("Accept", "application/json")
        if (token != null) connection.setRequestProperty("Authorization", "Bearer $token")
        if (key != null) connection.setRequestProperty("Idempotency-Key", key)
        try {
            if (method == "POST") {
                connection.doOutput = true
                connection.setRequestProperty("Content-Type", "application/json")
                connection.outputStream.use { it.write((body?.toString() ?: "{}").toByteArray(Charsets.UTF_8)) }
            }
            val status = connection.responseCode
            val stream = if (status >= 400) connection.errorStream else connection.inputStream
            val text = stream?.use {
                val limited = java.io.ByteArrayOutputStream()
                val bytes = ByteArray(8192)
                while (true) {
                    val read = it.read(bytes)
                    if (read < 0) break
                    if (limited.size() + read > 1_048_576) throw Problem("The BuyWise response was too large for mobile.")
                    limited.write(bytes, 0, read)
                }
                limited.toString("UTF-8")
            } ?: ""
            if (status in 300..399) throw Problem("BuyWise returned a redirect. Check the configured server URL.")
            return Result(status, text)
        } finally { connection.disconnect() }
    }

    private fun saveSession(result: Result, account: String? = null) {
        val value = JSONObject(result.text)
        val token = value.getString("token")
        check(Regex("[A-Za-z0-9_-]{43}").matches(token))
        Vault.update(context) { state ->
            if (account != null && state.optString("accountId") != account) {
                val keys = state.keys()
                while (keys.hasNext()) { keys.next(); keys.remove() }
                state.put("sessionId", UUID.randomUUID().toString())
                state.put("accountId", account)
            }
            state.put("token", token).put("expiresAt", System.currentTimeMillis() + value.getLong("expiresIn") * 1000)
        }
    }

    private fun expireSession() = Vault.update(context) { state ->
        state.remove("token")
        state.remove("expiresAt")
        state.remove("cache")
        state.put("captureEnabled", false)
    }

    private fun sessionRequest(path: String, method: String, body: JSONObject? = null, key: String? = null): Result {
        val state = Vault.read(context)
        var token = state.optString("token")
        if (token.isEmpty()) return Result(401, "{}")
        if (state.optLong("expiresAt") - System.currentTimeMillis() < 86_400_000) {
            val refresh = http("auth/refresh", "POST", token)
            if (refresh.ok) {
                saveSession(refresh)
                token = Vault.read(context).getString("token")
            } else if (refresh.status == 401) {
                expireSession()
                return refresh
            }
        }
        val response = http(path, method, token, body, key)
        if (response.status == 401) expireSession()
        return response
    }

    private fun result(response: Result, cached: Boolean = false) = JSObject()
        .put("status", response.status).put("cached", cached)
        .put("data", if (response.text.isBlank()) JSONObject.NULL else JSONTokener(response.text).nextValue())

    @PluginMethod fun status(call: PluginCall) = run(call) {
        JSObject().put("signedIn", Vault.read(context).optString("token").isNotEmpty())
    }

    @PluginMethod fun signIn(call: PluginCall) = run(call) {
        val body = JSONObject().put("email", call.getString("email")).put("password", call.getString("password"))
        val response = http("auth/login", "POST", body = body)
        if (!response.ok) throw Problem(if (response.status == 401) "Email or password is incorrect." else "Sign-in failed. Try again.")
        val session = http("auth/session", "GET", JSONObject(response.text).getString("token"))
        if (!session.ok) throw Problem("BuyWise could not verify this session. Try again.")
        val account = JSONObject(session.text).getString("id")
        check(Regex(uuid).matches(account))
        saveSession(response, account)
        JSObject()
    }

    @PluginMethod fun signOut(call: PluginCall) = run(call) {
        val token = Vault.read(context).optString("token")
        Vault.clear(context)
        if (token.isNotEmpty()) runCatching { http("auth/logout", "POST", token) }
        JSObject()
    }

    @PluginMethod fun request(call: PluginCall) = run(call) {
        val path = call.getString("path") ?: ""
        val method = call.getString("method") ?: "GET"
        val read = method == "GET" && readPaths.matches(path)
        if (!read && !(method == "POST" && writePaths.matches(path))) throw Problem("Unsupported mobile action.")
        val state = Vault.read(context)
        val snapshot = state.optJSONObject("cache")?.optJSONObject(path)
        val mayUseCache = read && state.optString("token").isNotEmpty() &&
            state.optLong("expiresAt") > System.currentTimeMillis() && snapshot != null &&
            System.currentTimeMillis() - snapshot.optLong("savedAt") < 86_400_000
        val response = try { sessionRequest(path, method, if (read) null else call.getObject("body"), call.getString("key")) }
            catch (err: IOException) {
                if (mayUseCache) return@run result(Result(200, snapshot!!.getString("body")), true)
                throw err
            }
        if (response.status >= 500 && mayUseCache) return@run result(Result(200, snapshot!!.getString("body")), true)
        if (response.ok) Vault.update(context) { saved ->
            if (read && response.text.length <= 96_000) {
                var entries = saved.optJSONObject("cache") ?: JSONObject()
                if (entries.length() >= 16 && !entries.has(path)) entries = JSONObject()
                entries.put(path, JSONObject().put("savedAt", System.currentTimeMillis()).put("body", response.text))
                saved.put("cache", entries)
            } else if (!read) saved.remove("cache")
        }
        result(response)
    }

    private fun granted(): Boolean {
        val expected = ComponentName(context, BuyWiseNotificationService::class.java)
        return Settings.Secure.getString(context.contentResolver, "enabled_notification_listeners")
            ?.split(':')?.any { ComponentName.unflattenFromString(it) == expected } ?: false
    }

    private fun captureState(rejected: Int = 0): JSObject {
        val state = Vault.read(context)
        return JSObject().put("granted", granted()).put("enabled", state.optBoolean("captureEnabled"))
            .put("packages", state.optJSONArray("packages") ?: JSONArray())
            .put("queued", state.optJSONArray("queue")?.length() ?: 0).put("rejected", rejected)
    }

    @PluginMethod fun captureStatus(call: PluginCall) = run(call) { captureState() }

    @PluginMethod fun configureCapture(call: PluginCall) = run(call) {
        val enabled = call.getBoolean("enabled") == true
        val input = call.getArray("packages") ?: JSONArray()
        val allowed = (0 until input.length()).map { input.optString(it) }.distinct()
        if (enabled && (!granted() || allowed.size !in 1..10 || allowed.any { it.length > 160 || !packageId.matches(it) }))
            throw Problem("Grant Android notification access and enter one to ten valid package IDs.")
        Vault.update(context) { state ->
            if (state.optString("token").isEmpty()) throw Problem("Sign in before setting up capture.")
            state.put("captureEnabled", enabled).put("packages", JSONArray(if (enabled) allowed else emptyList<String>()))
            val previous = state.optJSONArray("queue") ?: JSONArray()
            val queue = JSONArray()
            if (enabled) for (index in 0 until previous.length()) {
                val event = previous.getJSONObject(index)
                if (event.optString("sourceApp") in allowed) queue.put(event)
            }
            state.put("queue", queue)
        }
        captureState()
    }

    @PluginMethod fun openNotificationSettings(call: PluginCall) {
        try { activity.startActivity(Intent(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS)); call.resolve() }
        catch (_: Exception) { call.reject("Android notification settings are unavailable.") }
    }

    @PluginMethod fun sync(call: PluginCall) = run(call) {
        val starting = Vault.read(context)
        if (!granted() || !starting.optBoolean("captureEnabled")) throw Problem("Enable Android and BuyWise notification access before syncing.")
        var rejected = 0
        val count = starting.optJSONArray("queue")?.length() ?: 0
        repeat(minOf(count, 20)) {
            if (!granted()) throw Problem("Android notification access was removed.")
            val event = Vault.read(context).optJSONArray("queue")?.optJSONObject(0) ?: return@repeat
            val expired = Instant.parse(event.getString("occurredAt")).isBefore(Instant.now().minusSeconds(604_800))
            val response = if (expired) Result(400, "{}") else sessionRequest("expenses/candidates/notifications", "POST", event)
            if (response.status == 401) throw Problem("Your session ended. Sign in again.")
            if (!response.ok && response.status != 400) throw Problem("Sync paused. Unsent suggestions will be retried next time.")
            if (!response.ok) rejected++
            Vault.update(context) { state ->
                val queue = state.optJSONArray("queue")
                if (queue?.optJSONObject(0)?.toString() == event.toString()) queue.remove(0)
            }
        }
        captureState(rejected)
    }

    @PluginMethod fun openWeb(call: PluginCall) {
        val path = call.getString("path") ?: ""
        if (!Regex("/(?:sign-in|goals/$uuid)").matches(path)) { call.reject("Unsupported BuyWise page."); return }
        try { activity.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(server().resolve(path).toString()))); call.resolve() }
        catch (_: Exception) { call.reject("Unable to open BuyWise web.") }
    }
}
