# Datasource access with LBAC

This inert example gives two teams explicit `Query` access to one platform-owned datasource and
folds both teams' rules into the datasource's single authoritative LBAC rules tree. The datasource
UID is also the composite name, so Kubernetes admission permits only one owner for that UID in the
namespace.

An environment overlay must provide the datasource connection settings and replace the stack and
team identifiers with observed values. This public catalog contains no endpoint or credential.
The two forms of team identity are both required: datasource permissions use the numeric team ID,
while LBAC uses the team UID.

LBAC requires Grafana 11.5 or later, a Grafana Cloud or Enterprise entitlement, and a datasource
using basic authentication. The authoritative permission set retains only the declared team Query
grants among managed, non-inherited datasource permissions. Grafana permissions are additive, so
an inherited or independently managed broad query grant can still bypass LBAC and must be governed
outside this composite.
