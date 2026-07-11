import org.jetbrains.kotlin.gradle.plugin.mpp.apple.XCFramework

plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.android.library)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.sqldelight)
    alias(libs.plugins.skie)
}

kotlin {
    // JDK 17 toolchain for the JVM/Android compilation (matches :androidApp).
    jvmToolchain(17)

    // Android target: publish the release variant so :androidApp can depend on it.
    androidTarget {
        publishLibraryVariants("release")
    }

    // iOS targets. All three feed a single XCFramework named "Shared".
    val xcf = XCFramework("Shared")
    listOf(
        iosArm64(),
        iosSimulatorArm64(),
        iosX64(),
    ).forEach { iosTarget ->
        iosTarget.binaries.framework {
            baseName = "Shared"
            // Expose the whole shared module to Swift as one framework.
            isStatic = true
            xcf.add(this)
        }
    }

    sourceSets {
        commonMain.dependencies {
            implementation(libs.ktor.client.core)
            implementation(libs.ktor.client.content.negotiation)
            implementation(libs.ktor.serialization.kotlinx.json)
            implementation(libs.kotlinx.coroutines.core)
            implementation(libs.kotlinx.serialization.json)
            implementation(libs.kotlinx.datetime)
            implementation(libs.koin.core)
            implementation(libs.multiplatform.settings)
            implementation(libs.sqldelight.runtime)
            // Flow over SQLDelight queries: the diary list re-emits on every local
            // write, and SongRepository.observeSongHistory() streams play history.
            implementation(libs.sqldelight.coroutines.extensions)
        }

        androidMain.dependencies {
            implementation(libs.ktor.client.okhttp)
            implementation(libs.sqldelight.android.driver)
            implementation(libs.koin.android)
            // EncryptedSharedPreferences-backed SecureKeyValueStore.
            implementation(libs.androidx.security.crypto)
            // ExoPlayer backs the AudioPlayer actual on Android.
            implementation(libs.androidx.media3.exoplayer)
        }

        iosMain.dependencies {
            implementation(libs.ktor.client.darwin)
            implementation(libs.sqldelight.native.driver)
        }

        commonTest.dependencies {
            implementation(libs.kotlin.test)
            implementation(libs.ktor.client.mock)
            implementation(libs.kotlinx.coroutines.test)
        }

        // Shared repository tests run on the JVM (Android unit test) with the REAL
        // SQLDelight schema on a JVM in-memory driver + Ktor MockEngine — no
        // device/emulator needed. Both the diary sync engine and the today
        // repositories/player are exercised here. In androidUnitTest (not commonTest)
        // so no per-target test-driver actual is required.
        val androidUnitTest by getting {
            dependencies {
                implementation(libs.kotlin.test)
                implementation(libs.ktor.client.mock)
                implementation(libs.kotlinx.coroutines.test)
                implementation(libs.sqldelight.sqlite.driver)
            }
        }
    }
}

sqldelight {
    databases {
        create("RunaDatabase") {
            packageName.set("com.runa.shared.db")
        }
    }
}

android {
    namespace = "com.runa.shared"
    compileSdk = libs.versions.compileSdk.get().toInt()

    defaultConfig {
        minSdk = libs.versions.minSdk.get().toInt()
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}
