# HoleTab Roadmap

This document outlines the planned features and improvements for HoleTab. Since this is a solo project, priorities may shift based on personal needs and feedback.

## 🎯 Vision
A lightweight, dependency-free (no Node.js), and extremely fast self-hosted new-tab page that just works.

---

## 🟢 Phase 1: Core Enhancements (Short-term)
- [x] **Import/Export**: Support for importing from HTML bookmark files and exporting to HTML bookmark (Netscape Bookmark File Format).
- [x] **Drag-and-Drop Reordering**: Replace "up/down" buttons with a modern drag-and-drop interface (using HTMX or lightweight JS).
- [x] **Responsive Grid**: Improve the layout for mobile and tablet devices.
- [x] **Fix z-index issues**: Context menu be hides by other links in small screens.

## 🟡 Phase 2: User Experience (Mid-term)
- [x] **Custom Favicons**: Allow uploading custom icons or choosing from a set of predefined icons when auto-resolution fails.
- [x] **Weather Widget**: A minimal, privacy-focused weather display (optional/configurable).
- [ ] **Operating System Integration**: Support for Windows and macOS.
- [ ] **Organise**: organise links into folders and support HTML bookmarks file folder structure.

## 🔵 Phase 3: Technical & Infrastructure (Long-term)
- [ ] **Docker Support**: Provide a lightweight Scratch/Alpine-based Docker image.
- [ ] **Basic Auth**: Optional simple password protection for the web interface.
- [ ] **scripts**: move all scripts to a single directory for easier maintenance.
---

## ✅ Completed
- [x] Single-binary architecture.
- [x] Embedded static assets and templates.
- [x] Systemd integration via `install.sh`.
- [x] Automatic favicon resolution.
- [x] Persistent storage using `bbolt`.
