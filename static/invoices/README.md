# Invoice Backgrounds

The print template uses fixed coordinate canvases so text placement stays stable when the paper background changes.

- `hs-sales-invoice.png` is rendered into a `784 x 1032` canvas.
- `hs-charge-invoice.png` is rendered into a `1014 x 1214` canvas.
- `hs-stock-transfer-withdrawal.png` is rendered into a `718 x 848` canvas.

Replace the image files under the same names when the physical paper changes. The browser URL cache key is based on the image file modified time, so refreshing the invoice page will load the new background after replacement.

If a replacement scan has a different crop, angle, or margins, align/crop it to match the current image before replacing it. Otherwise the text coordinates will remain fixed but the printed paper artwork will move underneath them.
