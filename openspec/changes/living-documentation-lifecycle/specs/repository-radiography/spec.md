# Repository Radiography Specification

## Purpose

Define a read-only, evidence-based repository assessment before documentation planning or authoring.

## Requirements

### Requirement: Complete Evidence-Based Assessment

The system MUST produce a read-only radiography covering repository purpose, stack, modules, architecture, flows, data, execution, tests, deployment, risks, and discovered documentation. It SHALL attach supporting evidence or explicitly state that evidence is absent.

#### Scenario: Assess a populated repository

- GIVEN an accessible repository with source, configuration, and documentation
- WHEN a radiography is requested
- THEN the result covers every required assessment area with evidence or an absence statement

#### Scenario: Assess a repository with no documentation

- GIVEN an accessible repository containing no discovered documentation
- WHEN a radiography is requested
- THEN the result reports the empty documentation state without fabricating documents or claims

### Requirement: Discovery-Based Documentation Context

The system MUST discover candidate visible storage, audiences, language, and ownership from repository evidence. It MUST present candidates for confirmation and MUST NOT select a filename, location, language, or owner without that confirmation.

#### Scenario: Confirm discovered context

- GIVEN repository evidence identifies a documentation location and ownership signals
- WHEN radiography presents its findings
- THEN it presents candidates and their evidence for user confirmation

#### Scenario: Resolve conflicting ownership evidence

- GIVEN ownership signals conflict or do not identify an owner
- WHEN radiography is requested
- THEN it reports the conflict or unknown owner as uncertainty rather than choosing one

### Requirement: Explicit Limits and Exclusions

The system MUST identify excluded paths, incomplete evidence, confidence limits, and uncertain classifications. It MUST distinguish maintained documentation from generated, vendor, or otherwise excluded material when evidence supports that distinction.

#### Scenario: Exclude generated material

- GIVEN a repository contains generated or vendor documentation-like files
- WHEN radiography classifies discovered files
- THEN it records the exclusion and supporting evidence

#### Scenario: Report uncertain classification

- GIVEN evidence cannot determine whether a discovered file is maintained documentation
- WHEN radiography is requested
- THEN it marks the classification uncertain and records the missing or conflicting evidence
