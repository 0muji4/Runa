package com.runa.shared.core.state

import com.runa.shared.network.ApiException

/** 原因が特定できないときに見せる、ただ一つの総括文言。 */
const val GENERIC_ERROR_JA = "エラーが発生しました。"

/**
 * サーバー由来のメッセージを画面に素通しさせないためのマッピング層。
 *
 * バックエンドは機械可読な [ApiException.code]（apps/go internal/handler/response.go の
 * ErrorCode）を返すので、それを一次キーにする。英語のメッセージ本文は表示に使わない。
 * 未知の code、通信失敗（[ApiException] 以外）はすべて [fallback] に集約する。
 *
 * 呼び出し側は「その操作が失敗したときの日本語」を [fallback] に渡す
 * （例: 保存なら「保存できませんでした。」）。既定は [GENERIC_ERROR_JA]。
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
    // code が無いのはエラー封筒をパースできなかったときだけ。既知の英語本文だけ拾い、
    // それ以外は fallback。英語をそのまま返す経路は無い。
    val body = message?.lowercase().orEmpty()
    return when {
        body.contains("email or password is incorrect") -> "メールアドレスかパスワードが違います。"
        body.contains("email already registered") -> "このメールアドレスはすでに登録されています。"
        body.contains("validation failed") -> "入力内容を確認してください。"
        body.contains("too many requests") -> "アクセスが集中しています。少し待ってからもう一度ためしてください。"
        else -> fallback
    }
}
