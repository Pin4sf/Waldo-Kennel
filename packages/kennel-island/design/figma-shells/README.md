# Figma shell exports

These are the exported frame shells from the Waldo Figma file: the compact
shell, the expanded island, the notch outline, and the ticker plates.

They are kept as design reference only and are deliberately outside `public/`,
so they are not copied into a build.

The shipped island does not use them. Its shape is drawn in CSS
(`.island-body` and `.island-fillet` in `src/styles/app.css`) because a shape
that morphs between surfaces, matches the measured notch, and animates cannot
be a fixed picture. The icons and marks in `public/figma` are still exported
assets; only the chrome moved to CSS.
