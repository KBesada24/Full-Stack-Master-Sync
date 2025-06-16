import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, RefreshCcw } from "lucide-react";
import StatefulPart from "@/components/StatefulPart";

export default function FullStackSyncMaster() {
  //const serverData = getServerData();

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 to-slate-900 text-white p-6">
      <header className="text-center mb-10">
        <h1 className="text-4xl font-bold mb-2">Full Stack Sync Master</h1>
        <p className="text-slate-400">AI-powered sync and debugging tool for full stack developers</p>
      </header>

      <Tabs defaultValue="sync" className="max-w-4xl mx-auto">
        <TabsList className="grid grid-cols-3 bg-slate-800">
          <TabsTrigger value="sync">Sync Environments</TabsTrigger>
          <TabsTrigger value="ai">AI Suggestions</TabsTrigger>
          <TabsTrigger value="testing">E2E Tests</TabsTrigger>
        </TabsList>

        <TabsContent value="sync">
          <Card className="bg-slate-900 border-slate-700">
            <CardContent className="p-6">
              <h2 className="text-xl font-semibold mb-4">Environment Sync</h2>
              <p className="mb-4 text-slate-300">Connect your frontend and backend URLs:</p>
              <div className="space-y-4">
                <Input placeholder="Frontend URL (e.g. http://localhost:3000)" className="bg-slate-800 border-slate-600" />
                <Input placeholder="Backend URL (e.g. http://localhost:8080)" className="bg-slate-800 border-slate-600" />
                <Button className="w-full bg-blue-600 hover:bg-blue-700">Sync Now</Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="ai">
          <Card className="bg-slate-900 border-slate-700">
            <CardContent className="p-6">
              <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
                <Bot size={20} /> AI Code Suggestions
              </h2>
              <p className="text-slate-300 mb-4">Paste code snippets or describe the issue to get intelligent feedback:</p>
              <textarea
                className="w-full h-40 bg-slate-800 border border-slate-600 rounded-lg p-3 text-sm text-white"
                placeholder="Enter code or describe your issue here..."
              />
              <Button className="mt-4 bg-green-600 hover:bg-green-700 w-full">Get Suggestions</Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="testing">
          <Card className="bg-slate-900 border-slate-700">
            <CardContent className="p-6">
              <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
                <RefreshCcw size={20} /> Automated E2E Tests
              </h2>
              <p className="text-slate-300 mb-4">Run end-to-end tests to ensure consistency across your stack.</p>
              <Button className="bg-purple-600 hover:bg-purple-700 w-full">Run Tests</Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <footer className="text-center text-slate-500 mt-10 text-sm">
        &copy; 2025 Full Stack Sync Master. All rights reserved.
      </footer>
    </div>
  );
}

