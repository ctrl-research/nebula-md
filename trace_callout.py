#!/usr/bin/env python3
"""Trace goldmark output and parser behavior for multi-line callouts."""

import re

# Simulate what goldmark outputs for the multi-line callout test case
goldmark_output = """<blockquote>
<p>[!info]
Serving size: 5 meals<br />
Prep time: 10min<br />
Cooking time: 20min</p>
</blockquote>"""

print("=== Goldmark Output ===")
print(goldmark_output)
print()

# The typeRe pattern from parser.go
typeRe = re.compile(r'(?s)<blockquote>\s*<p>\[!([a-zA-Z]+)(\|([^\]\+]+))?\](\+|\-)?')

# Find the callout match
m = typeRe.search(goldmark_output)
if not m:
    print("ERROR: typeRe didn't match!")
    print("Looking for pattern:", typeRe.pattern)
else:
    print("=== typeRe Match ===")
    print(f"Match span: {m.span()}")
    print(f"Match group(0): {repr(m.group(0))}")
    print(f"Group 1 (type): {repr(m.group(1))}")
    print(f"Group 2 (|title): {repr(m.group(2))}")
    print(f"Group 3 (title): {repr(m.group(3))}")
    print(f"Group 4 (fold): {repr(m.group(4))}")
    print()

    typ = m.group(1).lower()
    print(f"typ = {repr(typ)}")

    # m.start(0) is start of full match, m.end(0) is end
    match_start = m.start()
    match_end = m.end()
    print(f"match_start = {match_start}, match_end = {match_end}")
    print(f"m.group(0) = {repr(m.group(0))}")
    print()

    # The inline content is between m.end() (after the ]) and the first </p>
    # Let's find the </p> in the matched text
    match_text = m.group(0)
    closeP_in_match = match_text.find("</p>")
    print(f"closeP_in_match = {closeP_in_match}")
    print()

    # After the ] in the match is where inline content starts
    # The ] is at the end of the full match group 0
    # Actually m.end() should be right after ]
    print(f"Content between m.end() and </p>: {repr(goldmark_output[match_end:match_start + closeP_in_match])}")

    # Now let's find the blockquote end
    bq_start = goldmark_output.find("<blockquote>")
    bq_end = goldmark_output.find("</blockquote>")
    print(f"\nbq_start = {bq_start}, bq_end = {bq_end}")

    # The actual closeP location in the full string
    actual_closeP = match_start + closeP_in_match
    print(f"actual_closeP = {actual_closeP}")
    print(f"Content at actual_closeP: {repr(goldmark_output[actual_closeP:actual_closeP+5])}")

    # Content starts after </p>
    contentStart = actual_closeP + 5
    print(f"\ncontentStart (after </p>) = {contentStart}")
    print(f"Content from contentStart to bq_end+len('</blockquote>'): {repr(goldmark_output[contentStart:bq_end + 13])}")

    # Now look for <p> tags in the remaining content
    remaining = goldmark_output[contentStart:bq_end + 13]
    print("\n=== Looking for <p> tags in remaining content ===")

    found_paragraphs = []
    while True:
        pStart = remaining.find("<p>")
        if pStart < 0:
            print("No more <p> tags")
            break
        remaining = remaining[pStart+3:]
        pEnd = remaining.find("</p>")
        if pEnd < 0:
            print("Unclosed <p>")
            break
        pText = remaining[:pEnd].rstrip("\n\r ")
        print(f"Found <p> with text: {repr(pText)}")
        found_paragraphs.append(pText)
        remaining = remaining[pEnd+4:]
        print(f"Remaining after: {repr(remaining[:50])}...")

    print()
    print("=== KEY INSIGHT ===")
    print(f"Number of <p> tags found: {len(found_paragraphs)}")
    print("If this is 1, it means goldmark puts ALL lines in ONE <p> with <br /> tags")
    print()

    # Simulate the fix - split by <br />
    print("=== Simulating the fix at lines 283-296 ===")
    if found_paragraphs:
        brRe = re.compile(r'(?i)<br\s*/?>')
        for i, pText in enumerate(found_paragraphs):
            print(f"\nOriginal pText[{i}]: {repr(pText)}")
            parts = brRe.split(pText, -1)
            print(f"After brRe.split: {parts}")

            for part in parts:
                part = part.rstrip("\n\r ")
                if part:
                    print(f"  -> Would create <p>{part}</p>")

    # The ACTUAL problem might be that the inline content extraction is wrong
    print()
    print("=== RE-EXAMINING THE CODE LOGIC ===")
    print("Looking at lines 262-264 of parser.go:")
    print("  inlineContent := strings.TrimSpace(string(htmlBody[m[1] : m[0]+closeP]))")
    print("where m[1] is end of type capture and m[0]+closeP is end of first </p>")
    print()
    print("m[1] = end of 'info' capture = end of 'info' in '[!info]'")
    print("So inlineContent would be: everything after 'info' until '</p>'")
    print()

    # Let me manually trace what the indices would be
    print("Indices in goldmark_output:")
    print(f"  '<blockquote>\\n<p>[!info]' starts at 0")
    print(f"  '[!info]' match is from index 12 to 21")
    print(f"  ']' is at index 20")
    print(f"  'm.end()' should be 21 (after ])")
    print()

    info_start = goldmark_output.find("[!info]")
    info_end = info_start + len("[!info]")
    print(f"info_start = {info_start}, info_end = {info_end}")
    print(f"goldmark_output[{info_start}:{info_end}] = {repr(goldmark_output[info_start:info_end])}")

    # The inline content
    inlineContent = goldmark_output[info_end:actual_closeP]
    print(f"\ninlineContent = goldmark_output[{info_end}:{actual_closeP}]")
    print(f"inlineContent = {repr(inlineContent)}")
    print(f"Trimmed: {repr(inlineContent.strip())}")

    # AHA! The issue: inlineContent would be "\nServing size: 5 meals<br />\nPrep time..."
    # because there's content on the SAME LINE as the header after ]

    print()
    print("=== THE REAL PROBLEM ===")
    print("The inline content extraction at line 264 captures content AFTER the ]")
    print("up to the first </p>. This INCLUDES the first line of content!")
    print()
    print("So 'Serving size: 5 meals<br />\\nPrep time: 10min<br />\\nCooking time: 20min'")
    print("gets captured as inlineContent, NOT left for the paragraph loop.")
    print()
    print("The fix needs to handle <br /> splitting in BOTH places:")
    print("  1. The paragraph loop (lines 283-296) - for content AFTER the first </p>")
    print("  2. The inlineContent handling (lines 305-326) - for content on the header line")