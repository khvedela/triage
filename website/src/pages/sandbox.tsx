import Layout from "@theme/Layout";
import SandboxLab from "../components/SandboxLab";

export default function SandboxPage(): JSX.Element {
  return (
    <Layout
      title="Sandbox | triage"
      description="Interactive scenario lab for triage Kubernetes diagnosis examples."
    >
      <main className="container margin-vert--lg">
        <SandboxLab />
      </main>
    </Layout>
  );
}
