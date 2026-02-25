import { UserCheck, Building2, Award, Search, Calendar, ShieldCheck } from 'lucide-react';

const individualFeatures = [
  { icon: Calendar, title: 'Attend Events', desc: 'Join tech meetups, medical workshops, and more.' },
  { icon: Award, title: 'Earn Badges', desc: 'Get verified credentials for completed events.' },
  { icon: UserCheck, title: 'Build Credibility', desc: 'Show employers proof of your real skills.' },
];

const orgFeatures = [
  { icon: Calendar, title: 'Post Events', desc: 'Create and manage events, internships, and jobs.' },
  { icon: Search, title: 'Discover Talent', desc: 'Find skilled participants from your event pool.' },
  { icon: ShieldCheck, title: 'Verify Participation', desc: 'Confirm attendance and issue skill badges.' },
];

function FeatureCard({ icon: Icon, title, desc }) {
  return (
    <div className="flex items-start gap-4 rounded-xl bg-background p-5 card-shadow">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-accent">
        <Icon className="h-5 w-5 text-accent-foreground" />
      </div>
      <div>
        <h3 className="mb-1 font-semibold text-foreground">{title}</h3>
        <p className="text-sm text-muted-foreground">{desc}</p>
      </div>
    </div>
  );
}

export default function WhoThisIsFor() {
  return (
    <div className="section-padding">
      <div className="container max-w-5xl">
        <h1 className="mb-4 text-center text-4xl font-bold text-foreground md:text-5xl">
          Who Is <span className="text-gradient">SkillZone</span> For?
        </h1>
        <p className="mx-auto mb-16 max-w-2xl text-center text-lg text-muted-foreground">
          Whether you're building your career or discovering top talent, SkillZone has you covered.
        </p>

        <div className="grid gap-12 lg:grid-cols-2">
          {/* Individuals */}
          <div className="rounded-2xl bg-card p-8 card-shadow">
            <div className="mb-6 flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary">
                <UserCheck className="h-6 w-6 text-primary-foreground" />
              </div>
              <h2 className="text-2xl font-bold text-foreground">For Individuals</h2>
            </div>
            <div className="space-y-4">
              {individualFeatures.map((f, i) => (
                <FeatureCard key={i} {...f} />
              ))}
            </div>
          </div>

          {/* Organizations */}
          <div className="rounded-2xl bg-card p-8 card-shadow">
            <div className="mb-6 flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary">
                <Building2 className="h-6 w-6 text-primary-foreground" />
              </div>
              <h2 className="text-2xl font-bold text-foreground">For Organizations</h2>
            </div>
            <div className="space-y-4">
              {orgFeatures.map((f, i) => (
                <FeatureCard key={i} {...f} />
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
