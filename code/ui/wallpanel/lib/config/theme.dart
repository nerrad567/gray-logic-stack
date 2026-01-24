import 'package:flutter/material.dart';

/// Dark theme optimised for always-on wall panel displays.
/// Reduces glare, uses warm amber for active controls.
ThemeData wallPanelTheme() {
  const background = Color(0xFF1A1A2E);
  const surface = Color(0xFF2D2D44);
  const primary = Color(0xFFF5A623);
  const secondary = Color(0xFF4ECDC4);
  const onSurface = Color(0xFFE0E0E0);
  const error = Color(0xFFFF6B6B);

  return ThemeData(
    brightness: Brightness.dark,
    scaffoldBackgroundColor: background,
    colorScheme: const ColorScheme.dark(
      primary: primary,
      secondary: secondary,
      surface: surface,
      onSurface: onSurface,
      error: error,
    ),
    textTheme: const TextTheme(
      titleLarge: TextStyle(
        fontSize: 24,
        fontWeight: FontWeight.w300,
        color: onSurface,
      ),
      titleMedium: TextStyle(
        fontSize: 16,
        fontWeight: FontWeight.w500,
        color: onSurface,
      ),
      bodyLarge: TextStyle(
        fontSize: 14,
        color: Color(0xFFB0B0B0),
      ),
      bodyMedium: TextStyle(
        fontSize: 12,
        color: Color(0xFF808080),
      ),
    ),
    cardTheme: CardThemeData(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
      ),
      color: surface,
    ),
    useMaterial3: true,
  );
}
