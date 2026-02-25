import { Target, Eye, ShieldCheck } from 'lucide-react';

const sections = [
  {
    icon: Target,
    title: 'Our Mission',
    body: 'To empower individuals with verifiable skill credentials and connect them with meaningful opportunities — making talent discovery transparent and trustworthy.',
  },
  {
    icon: Eye,
    title: 'Our Vision',
    body: 'A world where skills speak louder than resumes. We envision a future where every individual can prove their abilities through verified, badge-backed experiences.',
  },
  {
    icon: ShieldCheck,
    title: 'Why Skill Verification Matters',
    body: 'Traditional resumes can be embellished. SkillZone ensures every badge is earned through real participation, verified by the hosting organization. Employers get confidence; individuals get credibility.',
  },
];

export default function About() {
  return (
    <div className="section-padding">
      <div className="container max-w-4xl">
        <h1 className="mb-4 text-center text-4xl font-bold text-foreground md:text-5xl">
          About <span className="text-gradient">SkillZone</span>
        </h1>
        <p className="mx-auto mb-16 max-w-2xl text-center text-lg text-muted-foreground">
          SkillZone is a platform designed to bridge the trust gap between job seekers and
          employers through verified skill credentials earned from real-world events and
          experiences.
        </p>

        <div className="space-y-10">
          {sections.map((s, i) => (
            <div
              key={i}
              className="card-shadow flex gap-6 rounded-2xl bg-card p-8 transition hover:card-shadow-hover"
            >
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-accent">
                <s.icon className="h-6 w-6 text-accent-foreground" />
              </div>
              <div>
                <h2 className="mb-2 text-xl font-semibold text-foreground">{s.title}</h2>
                <p className="leading-relaxed text-muted-foreground">{s.body}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
