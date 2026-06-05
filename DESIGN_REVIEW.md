# Design Review: Smart Edu

Reviewed against: no `DESIGN_BRIEF.md` found in repo; reviewed against current product intent and built UI
Philosophy: practical, desktop-first AI simulation UI
Date: 2026-06-05

## Screenshots Captured

| Screenshot | Breakpoint | Description |
| --- | --- | --- |
| `C:\Users\Shoxjaxon\Desktop\smart-edu\design-review-desktop-1280.png` | Desktop (1280×800) | Student home with chat + terminal split view |
| `C:\Users\Shoxjaxon\Desktop\smart-edu\design-review-tablet-768.png` | Tablet (768×1024) | Student home at medium width |
| `C:\Users\Shoxjaxon\Desktop\smart-edu\design-review-student-mobile-375.png` | Mobile (375×812) | Student home with stacked mobile tabs |
| `C:\Users\Shoxjaxon\Desktop\smart-edu\design-review-teacher-desktop-1280.png` | Desktop (1280×800) | Teacher upload/editor dashboard |
| `C:\Users\Shoxjaxon\Desktop\smart-edu\design-review-teacher-mobile-375.png` | Mobile (375×812) | Teacher dashboard stacked for mobile |

## Summary

The UI has a clear product idea and the desktop student view reads well: strong split-pane structure, clear state chips, and usable spacing. The biggest problem is the teacher experience: it surfaces a raw `HTTP 404` error banner in a prominent spot, which makes the page feel broken on load. Mobile also needs tighter information architecture, especially on the teacher view where the core editor gets pushed far below the first screen.

## Must Fix

1. **Raw `HTTP 404` banner in teacher panel**: the teacher page shows a top-level error state with the literal transport error text instead of a user-facing empty/error state. See [`frontend/src/components/TeacherPanel.vue`](C:\Users\Shoxjaxon\Desktop\smart-edu\frontend\src\components\TeacherPanel.vue#L84) and [`design-review-teacher-desktop-1280.png`](C:\Users\Shoxjaxon\Desktop\smart-edu\design-review-teacher-desktop-1280.png). _Fix: map API failures to a friendly empty state or action-oriented message, and make sure the teacher endpoints are actually returning expected data before surfacing the panel._

## Should Fix

1. **Teacher mobile layout is too vertically deep**: on 375px width, the upload card dominates the first viewport and the documents/editor/scenario lists are pushed well below the fold. See [`frontend/src/components/TeacherPanel.vue`](C:\Users\Shoxjaxon\Desktop\smart-edu\frontend\src\components\TeacherPanel.vue#L299) and [`design-review-teacher-mobile-375.png`](C:\Users\Shoxjaxon\Desktop\smart-edu\design-review-teacher-mobile-375.png). _Fix: collapse secondary panes into tabs/accordions on mobile, or reduce vertical chrome so the editor is reachable without long scrolling._

2. **Duplicate navigation layers on mobile**: the global Student/Teacher nav plus the in-panel Chat/Terminal toggle creates too much chrome at the top of the student experience. See [`frontend/src/App.vue`](C:\Users\Shoxjaxon\Desktop\smart-edu\frontend\src\App.vue#L31) and [`frontend/src/components/ChatPanel.vue`](C:\Users\Shoxjaxon\Desktop\smart-edu\frontend\src\components\ChatPanel.vue#L1). _Fix: keep one primary mode switch at a time, or visually merge the navigation so it reads as a single system._

## Could Improve

1. **Visual identity is functional but generic**: the beige grid background helps a bit, but the type stack and component styling still read like a safe default admin UI rather than a distinctive product. _Suggestion: tighten the brand system with a more intentional headline font, stronger accent usage, and one or two signature surfaces so the product has a clearer visual character._

2. **Error styling is too close to the neutral palette**: the teacher error chip is easy to miss as a state, even though it is important. _Suggestion: reserve a more deliberate semantic color and add an icon or title so failures read immediately._

## What Works Well

- Desktop student layout is structurally sound: the chat/terminal split gives the product a clear workflow, and the hierarchy is easy to follow.
- Responsive behavior is already present in both student and teacher panels, and the mobile navigation does adapt rather than simply shrinking the desktop layout.
- Spacing and radii are mostly consistent across panels, cards, and inputs, which keeps the interface coherent.
