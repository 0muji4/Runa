package com.runa.shared.core.state

import com.runa.shared.network.ApiException

/** 原因が特定できないときに見せる、ただ一つの総括文言。 */
const val GENERIC_ERROR_JA = "エラーが発生しました。"

/**
 * [ApiException.code] を一次キーに日本語文言へ写す。未知の code・通信失敗は [fallback]
 * （その操作が失敗したときの日本語。既定は [GENERIC_ERROR_JA]）。
 */
fun Throwable.toJaMessage(fallback: String = GENERIC_ERROR_JA): String {
    val api = this as? ApiException ?: return fallback
    api.code?.let { code ->
        return when (code) {
            "invalid_credentials" -> "メールアドレスかパスワードが違います。"
            "email_taken" -> "このメールアドレスはすでに登録されています。"
            "validation_error" -> "入力内容を確認してください。"
            "rate_limited" -> "アクセスが集中しています。少し待ってからもう一度ためしてください。"
            else -> fallback
        }
    }
    // code が無い（封筒をパースできなかった）ときも英語本文をそのまま返す経路は無い。
    val body = message?.lowercase().orEmpty()
    return when {
        body.contains("email or password is incorrect") -> "メールアドレスかパスワードが違います。"
        body.contains("email already registered") -> "このメールアドレスはすでに登録されています。"
        body.contains("validation failed") -> "入力内容を確認してください。"
        body.contains("too many requests") -> "アクセスが集中しています。少し待ってからもう一度ためしてください。"
        else -> fallback
    }
}
