# Changelog

## 1.1.7

- Replaced dropdown option rows that were sibling child HWNDs with dedicated owned `WS_POPUP` windows.
- Operation and Language dropdowns now always render above Source/Output edit controls, even while those edits repaint or receive focus.
- Preserved fully custom dark dropdown rows, single chevron, hover animation, Turkish/English labels and Arial UI.

## 1.1.6

- Replaced native Windows ComboBox controls with fully custom dark selectors.
- Removed duplicate/native Windows dropdown arrow buttons.
- Added single integrated chevron, custom hover/focus styling and fully custom dropdown option rows.
- Turkish / English and operation selectors now use the same visual system as the rest of HDR Studio.
- Preserved DXT-only extraction, batch pack/unpack, Arial UI and flicker-free progress rendering.

## 1.1.4

- Fixed dynamic source/output label ghosting when changing Pack/Unpack modes.
- Added opaque repainting for dark-mode labels to prevent stale text artifacts.
- Added 60 FPS lightweight hover/focus animation for action buttons, checkboxes and selectors.
- Added custom dark animated arrow buttons over the native operation/language selector arrows.
- Increased field spacing and padding around the extracted-files root section.
- Refined button hover borders and dark-mode contrast while keeping Arial throughout the UI.

## 1.1.3

- Fixed owner-drawn checkbox labels accumulating/overlapping after mode or language changes.
- Checkbox rows now clear their background before each redraw.
- Switched the complete interface font family to Arial.
- Increased checkbox spacing and hit area for a cleaner dark-mode layout.
- Checkbox labels are rendered in solid white with a refined rounded checkbox style.

## 1.1.2
- Replaced native checkbox rendering with fully owner-drawn dark-mode checkboxes.
- Checkbox labels are now bright white in dark mode.
- Checked state uses the application blue accent and a white tick.
- Options that do not apply to the selected operation are hidden completely, so no empty checkbox remains on screen.
- Preserved Batch Unpack forced archive-retention behavior while keeping its label readable.

## 1.1.1
- Improved dark-mode text handling.

## 1.1.0
- Added selective `.dxt`-only extraction for single and batch unpack.
- Added recursive Batch Unpack and Batch Pack.
- Added default batch root `C:\\NTTGame\\KnightOnlineEn`.
- Added optional external output / extracted working root while preserving relative folder layout.
- Batch Unpack always preserves source HDR/SRC pairs.
- Single Pack honors the optional archive output directory.
- Added Turkish / English labels for the new modes.
