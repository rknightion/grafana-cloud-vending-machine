# Project content

This inert request demonstrates the platform-owned profile seam for bounded content across an
existing project stack and central stack. The selected profile must authorize the namespace, both
stack identities, exact project-label equality, and the allowed content set before reconciliation.

Replace `example-project-content` with a reviewed platform profile. The request does not contain
provider credentials or enable itself; the supplied Argo CD ApplicationSet watches only `enabled/`.
