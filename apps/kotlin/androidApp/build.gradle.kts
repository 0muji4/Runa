plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
}

android {
    namespace = "com.runa.android"
    compileSdk = libs.versions.compileSdk.get().toInt()

    defaultConfig {
        applicationId = "com.runa"
        minSdk = libs.versions.minSdk.get().toInt()
        targetSdk = libs.versions.targetSdk.get().toInt()
        versionCode = 1
        versionName = "0.1.0"

        // Dev base URL for the Android emulator: 10.0.2.2 is the host loopback.
        // host+port ONLY — the shared module appends /api/v1/healthz itself.
        // TODO: switch per build type / flavor once a staging/prod host exists.
        buildConfigField("String", "BASE_URL", "\"http://10.0.2.2:8080\"")

        // Native sign-in configuration (see README). Empty by default so the app
        // builds without credentials; the Google/Apple buttons surface a clear
        // error until these are filled in.
        //  - GOOGLE_SERVER_CLIENT_ID: the Google OAuth *Web* client ID, used both
        //    as Credential Manager's serverClientId and the backend audience.
        //  - APPLE_SERVICE_ID / APPLE_REDIRECT_URI: the Apple Service ID and its
        //    https redirect for the Sign in with Apple web flow on Android.
        buildConfigField("String", "GOOGLE_SERVER_CLIENT_ID", "\"${project.findProperty("RUNA_GOOGLE_SERVER_CLIENT_ID") ?: ""}\"")
        buildConfigField("String", "APPLE_SERVICE_ID", "\"${project.findProperty("RUNA_APPLE_SERVICE_ID") ?: ""}\"")
        buildConfigField("String", "APPLE_REDIRECT_URI", "\"${project.findProperty("RUNA_APPLE_REDIRECT_URI") ?: ""}\"")
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildTypes {
        release {
            isMinifyEnabled = false
        }
    }
}

dependencies {
    implementation(project(":shared"))

    val composeBom = platform(libs.compose.bom)
    implementation(composeBom)
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.tooling.preview)
    implementation(libs.compose.material3)
    debugImplementation(libs.compose.ui.tooling)

    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.navigation.compose)
    implementation(libs.androidx.lifecycle.viewmodel.compose)

    implementation(libs.koin.android)
    implementation(libs.koin.androidx.compose)

    // Native sign-in: Google via Credential Manager, Apple via a Custom Tabs web flow.
    implementation(libs.androidx.credentials)
    implementation(libs.androidx.credentials.play.services.auth)
    implementation(libs.google.identity.googleid)
    implementation(libs.androidx.browser)
}
