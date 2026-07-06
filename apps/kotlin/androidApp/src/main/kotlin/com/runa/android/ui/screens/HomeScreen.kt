package com.runa.android.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.runa.android.R
import com.runa.shared.feature.health.HealthzUiState
import com.runa.shared.feature.health.HealthzViewModel
import androidx.compose.runtime.collectAsState
import org.koin.compose.koinInject

/**
 * Home tab. The only screen wired to a shared feature: it subscribes to
 * [HealthzViewModel] (resolved from Koin) and reflects the backend health probe.
 * Settings is reachable from the top-bar action.
 *
 * TODO: real Home content (walking summary, moon phase, ...) goes below the
 * status line as the Home vertical slice is built out.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    onSettingsClick: () -> Unit,
    viewModel: HealthzViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.tab_home)) },
                actions = {
                    TextButton(onClick = onSettingsClick) {
                        Text(stringResource(R.string.tab_settings))
                    }
                },
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 32.dp, vertical = 48.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp, Alignment.CenterVertically),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            when (val current = state) {
                is HealthzUiState.Loading -> CircularProgressIndicator()
                is HealthzUiState.Ok -> Text(
                    text = stringResource(R.string.health_ok),
                    style = MaterialTheme.typography.titleLarge,
                )
                is HealthzUiState.Error -> {
                    Text(
                        text = stringResource(R.string.health_error),
                        style = MaterialTheme.typography.titleLarge,
                        color = MaterialTheme.colorScheme.primary,
                    )
                    Text(
                        text = current.message,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }
}
